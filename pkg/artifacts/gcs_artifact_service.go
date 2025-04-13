// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package artifacts

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GcsArtifactService provides an implementation of the ArtifactService
// that uses Google Cloud Storage for persisting artifacts.
type GcsArtifactService struct {
	bucketName    string
	storageClient *storage.Client
	bucket        *storage.BucketHandle
}

// NewGcsArtifactService creates a new instance of GcsArtifactService.
func NewGcsArtifactService(ctx context.Context, bucketName string) (*GcsArtifactService, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %v", err)
	}

	return &GcsArtifactService{
		bucketName:    bucketName,
		storageClient: client,
		bucket:        client.Bucket(bucketName),
	}, nil
}

// fileHasUserNamespace checks if the filename has a user namespace.
func (s *GcsArtifactService) fileHasUserNamespace(filename string) bool {
	return strings.HasPrefix(filename, "user:")
}

// getBlobName constructs the blob name in GCS.
func (s *GcsArtifactService) getBlobName(appName, userID, sessionID, filename string, version int) string {
	if s.fileHasUserNamespace(filename) {
		return fmt.Sprintf("%s/%s/user/%s/%d", appName, userID, filename, version)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%d", appName, userID, sessionID, filename, version)
}

// SaveArtifact saves an artifact to Google Cloud Storage.
func (s *GcsArtifactService) SaveArtifact(ctx context.Context, appName, userID, sessionID, filename string, artifact Part) (int, error) {
	versions, err := s.ListVersions(ctx, appName, userID, sessionID, filename)
	if err != nil {
		return 0, fmt.Errorf("failed to list versions: %v", err)
	}

	version := 0
	if len(versions) > 0 {
		version = versions[len(versions)-1] + 1
	}

	blobName := s.getBlobName(appName, userID, sessionID, filename, version)
	obj := s.bucket.Object(blobName)
	w := obj.NewWriter(ctx)

	// Set content type
	w.ContentType = artifact.MimeType

	if len(artifact.Data) > 0 {
		if _, err := w.Write(artifact.Data); err != nil {
			w.Close()
			return 0, fmt.Errorf("failed to write data to GCS: %v", err)
		}
	} else if artifact.Text != "" {
		if _, err := w.Write([]byte(artifact.Text)); err != nil {
			w.Close()
			return 0, fmt.Errorf("failed to write text to GCS: %v", err)
		}
	}

	if err := w.Close(); err != nil {
		return 0, fmt.Errorf("failed to close GCS writer: %v", err)
	}

	return version, nil
}

// LoadArtifact retrieves an artifact from Google Cloud Storage.
func (s *GcsArtifactService) LoadArtifact(ctx context.Context, appName, userID, sessionID, filename string, version *int) (*Part, error) {
	var versionToLoad int

	if version == nil {
		versions, err := s.ListVersions(ctx, appName, userID, sessionID, filename)
		if err != nil {
			return nil, fmt.Errorf("failed to list versions: %v", err)
		}
		if len(versions) == 0 {
			return nil, nil
		}
		versionToLoad = versions[len(versions)-1]
	} else {
		versionToLoad = *version
	}

	blobName := s.getBlobName(appName, userID, sessionID, filename, versionToLoad)
	obj := s.bucket.Object(blobName)

	attrs, err := obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get object attributes: %v", err)
	}

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create object reader: %v", err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %v", err)
	}

	part := Part{
		Data:     data,
		MimeType: attrs.ContentType,
	}

	return &part, nil
}

// ListArtifactKeys lists all artifact filenames within a session.
func (s *GcsArtifactService) ListArtifactKeys(ctx context.Context, appName, userID, sessionID string) ([]string, error) {
	filenames := make(map[string]struct{})

	// List session artifacts
	sessionPrefix := fmt.Sprintf("%s/%s/%s/", appName, userID, sessionID)
	it := s.bucket.Objects(ctx, &storage.Query{Prefix: sessionPrefix})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating objects: %v", err)
		}

		parts := strings.Split(attrs.Name, "/")
		if len(parts) >= 5 { // app/user/session/filename/version
			filenames[parts[3]] = struct{}{}
		}
	}

	// List user namespace artifacts
	userNamespacePrefix := fmt.Sprintf("%s/%s/user/", appName, userID)
	it = s.bucket.Objects(ctx, &storage.Query{Prefix: userNamespacePrefix})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating objects: %v", err)
		}

		parts := strings.Split(attrs.Name, "/")
		if len(parts) >= 4 { // app/user/user/filename/version
			filenames[parts[3]] = struct{}{}
		}
	}

	result := make([]string, 0, len(filenames))
	for filename := range filenames {
		result = append(result, filename)
	}

	sort.Strings(result)
	return result, nil
}

// DeleteArtifact removes all versions of an artifact from Google Cloud Storage.
func (s *GcsArtifactService) DeleteArtifact(ctx context.Context, appName, userID, sessionID, filename string) error {
	versions, err := s.ListVersions(ctx, appName, userID, sessionID, filename)
	if err != nil {
		return fmt.Errorf("failed to list versions: %v", err)
	}

	for _, version := range versions {
		blobName := s.getBlobName(appName, userID, sessionID, filename, version)
		obj := s.bucket.Object(blobName)
		if err := obj.Delete(ctx); err != nil && err != storage.ErrObjectNotExist {
			return fmt.Errorf("failed to delete object %s: %v", blobName, err)
		}
	}

	return nil
}

// ListVersions lists all versions of an artifact.
func (s *GcsArtifactService) ListVersions(ctx context.Context, appName, userID, sessionID, filename string) ([]int, error) {
	prefix := s.getBlobName(appName, userID, sessionID, filename, -1)
	prefix = prefix[:strings.LastIndex(prefix, "/")+1]

	it := s.bucket.Objects(ctx, &storage.Query{Prefix: prefix})

	var versions []int
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating objects: %v", err)
		}

		parts := strings.Split(attrs.Name, "/")
		if len(parts) > 0 {
			versionStr := parts[len(parts)-1]
			version, err := strconv.Atoi(versionStr)
			if err != nil {
				continue
			}
			versions = append(versions, version)
		}
	}

	sort.Ints(versions)
	return versions, nil
}
