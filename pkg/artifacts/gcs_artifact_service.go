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
	"io"
	"sort"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GcsArtifactService provides a Google Cloud Storage implementation of the ArtifactService.
type GcsArtifactService struct {
	bucketName    string
	storageClient *storage.Client
	bucket        *storage.BucketHandle
}

// NewGcsArtifactService creates a new instance of GcsArtifactService.
func NewGcsArtifactService(ctx context.Context, bucketName string, opts ...option.ClientOption) (*GcsArtifactService, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
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
		return appName + "/" + userID + "/user/" + filename + "/" + strconv.Itoa(version)
	}
	return appName + "/" + userID + "/" + sessionID + "/" + filename + "/" + strconv.Itoa(version)
}

// SaveArtifact saves an artifact to Google Cloud Storage.
func (s *GcsArtifactService) SaveArtifact(ctx context.Context, appName, userID, sessionID, filename string, artifact Part) (int, error) {
	versions, err := s.ListVersions(ctx, appName, userID, sessionID, filename)
	if err != nil {
		return 0, err
	}

	version := 0
	if len(versions) > 0 {
		// Find max version and increment
		maxVersion := versions[0]
		for _, v := range versions {
			if v > maxVersion {
				maxVersion = v
			}
		}
		version = maxVersion + 1
	}

	blobName := s.getBlobName(appName, userID, sessionID, filename, version)
	obj := s.bucket.Object(blobName)
	w := obj.NewWriter(ctx)
	w.ContentType = artifact.MimeType

	if len(artifact.Data) > 0 {
		if _, err := w.Write(artifact.Data); err != nil {
			w.Close()
			return 0, err
		}
	} else if artifact.Text != "" {
		if _, err := w.Write([]byte(artifact.Text)); err != nil {
			w.Close()
			return 0, err
		}
	}

	if err := w.Close(); err != nil {
		return 0, err
	}

	return version, nil
}

// LoadArtifact retrieves an artifact from Google Cloud Storage.
func (s *GcsArtifactService) LoadArtifact(ctx context.Context, appName, userID, sessionID, filename string, version *int) (*Part, error) {
	var v int
	var err error

	if version == nil {
		versions, err := s.ListVersions(ctx, appName, userID, sessionID, filename)
		if err != nil {
			return nil, err
		}
		if len(versions) == 0 {
			return nil, nil
		}

		// Find max version
		v = versions[0]
		for _, ver := range versions {
			if ver > v {
				v = ver
			}
		}
	} else {
		v = *version
	}

	blobName := s.getBlobName(appName, userID, sessionID, filename, v)
	obj := s.bucket.Object(blobName)

	attrs, err := obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	part := &Part{
		MimeType: attrs.ContentType,
	}

	// Decide whether to store as text or binary based on content type
	if strings.HasPrefix(attrs.ContentType, "text/") ||
		attrs.ContentType == "application/json" ||
		attrs.ContentType == "application/xml" {
		part.Text = string(data)
	} else {
		part.Data = data
	}

	return part, nil
}

// ListArtifactKeys lists all artifact filenames within a session.
func (s *GcsArtifactService) ListArtifactKeys(ctx context.Context, appName, userID, sessionID string) ([]string, error) {
	filenamesMap := make(map[string]struct{})

	// List session-specific artifacts
	sessionPrefix := appName + "/" + userID + "/" + sessionID + "/"
	it := s.bucket.Objects(ctx, &storage.Query{Prefix: sessionPrefix})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		parts := strings.Split(attrs.Name, "/")
		if len(parts) >= 4 {
			filename := parts[3]
			filenamesMap[filename] = struct{}{}
		}
	}

	// List user namespace artifacts
	userNamespacePrefix := appName + "/" + userID + "/user/"
	it = s.bucket.Objects(ctx, &storage.Query{Prefix: userNamespacePrefix})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		parts := strings.Split(attrs.Name, "/")
		if len(parts) >= 4 {
			filename := parts[3]
			filenamesMap[filename] = struct{}{}
		}
	}

	// Convert map to sorted slice
	filenames := make([]string, 0, len(filenamesMap))
	for filename := range filenamesMap {
		filenames = append(filenames, filename)
	}

	sort.Strings(filenames)
	return filenames, nil
}

// DeleteArtifact removes an artifact from Google Cloud Storage.
func (s *GcsArtifactService) DeleteArtifact(ctx context.Context, appName, userID, sessionID, filename string) error {
	versions, err := s.ListVersions(ctx, appName, userID, sessionID, filename)
	if err != nil {
		return err
	}

	for _, version := range versions {
		blobName := s.getBlobName(appName, userID, sessionID, filename, version)
		obj := s.bucket.Object(blobName)
		if err := obj.Delete(ctx); err != nil && err != storage.ErrObjectNotExist {
			return err
		}
	}

	return nil
}

// ListVersions lists all versions of an artifact.
func (s *GcsArtifactService) ListVersions(ctx context.Context, appName, userID, sessionID, filename string) ([]int, error) {
	prefix := s.getBlobName(appName, userID, sessionID, filename, 0)
	prefix = prefix[:len(prefix)-1] // Remove the trailing "0"

	it := s.bucket.Objects(ctx, &storage.Query{Prefix: prefix})

	var versions []int
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		parts := strings.Split(attrs.Name, "/")
		if len(parts) >= 5 {
			versionStr := parts[4]
			version, err := strconv.Atoi(versionStr)
			if err == nil {
				versions = append(versions, version)
			}
		}
	}

	sort.Ints(versions)
	return versions, nil
}

// Close closes the GCS client.
func (s *GcsArtifactService) Close() error {
	return s.storageClient.Close()
}
