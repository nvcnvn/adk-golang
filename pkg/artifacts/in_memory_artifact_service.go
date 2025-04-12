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
	"sort"
	"strings"
	"sync"
)

// InMemoryArtifactService provides an in-memory implementation of the ArtifactService.
type InMemoryArtifactService struct {
	artifacts map[string][]Part
	mu        sync.RWMutex
}

// NewInMemoryArtifactService creates a new instance of InMemoryArtifactService.
func NewInMemoryArtifactService() *InMemoryArtifactService {
	return &InMemoryArtifactService{
		artifacts: make(map[string][]Part),
	}
}

// fileHasUserNamespace checks if the filename has a user namespace.
func (s *InMemoryArtifactService) fileHasUserNamespace(filename string) bool {
	return strings.HasPrefix(filename, "user:")
}

// artifactPath constructs the artifact path based on the provided parameters.
func (s *InMemoryArtifactService) artifactPath(appName, userID, sessionID, filename string) string {
	if s.fileHasUserNamespace(filename) {
		return appName + "/" + userID + "/user/" + filename
	}
	return appName + "/" + userID + "/" + sessionID + "/" + filename
}

// SaveArtifact saves an artifact to the in-memory storage.
func (s *InMemoryArtifactService) SaveArtifact(ctx context.Context, appName, userID, sessionID, filename string, artifact Part) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.artifactPath(appName, userID, sessionID, filename)
	if _, exists := s.artifacts[path]; !exists {
		s.artifacts[path] = []Part{}
	}

	version := len(s.artifacts[path])
	s.artifacts[path] = append(s.artifacts[path], artifact)
	return version, nil
}

// LoadArtifact retrieves an artifact from the in-memory storage.
func (s *InMemoryArtifactService) LoadArtifact(ctx context.Context, appName, userID, sessionID, filename string, version *int) (*Part, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.artifactPath(appName, userID, sessionID, filename)
	versions, exists := s.artifacts[path]
	if !exists || len(versions) == 0 {
		return nil, nil
	}

	var idx int
	if version == nil {
		idx = len(versions) - 1 // Latest version
	} else {
		idx = *version
		if idx < 0 || idx >= len(versions) {
			return nil, nil
		}
	}

	result := versions[idx]
	return &result, nil
}

// ListArtifactKeys lists all artifact filenames within a session.
func (s *InMemoryArtifactService) ListArtifactKeys(ctx context.Context, appName, userID, sessionID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionPrefix := appName + "/" + userID + "/" + sessionID + "/"
	usernamespacePrefix := appName + "/" + userID + "/user/"

	var filenames []string
	for path := range s.artifacts {
		if strings.HasPrefix(path, sessionPrefix) {
			filename := strings.TrimPrefix(path, sessionPrefix)
			filenames = append(filenames, filename)
		} else if strings.HasPrefix(path, usernamespacePrefix) {
			filename := strings.TrimPrefix(path, usernamespacePrefix)
			filenames = append(filenames, filename)
		}
	}

	sort.Strings(filenames)
	return filenames, nil
}

// DeleteArtifact removes an artifact from the in-memory storage.
func (s *InMemoryArtifactService) DeleteArtifact(ctx context.Context, appName, userID, sessionID, filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.artifactPath(appName, userID, sessionID, filename)
	if _, exists := s.artifacts[path]; !exists {
		return nil
	}

	delete(s.artifacts, path)
	return nil
}

// ListVersions lists all versions of an artifact.
func (s *InMemoryArtifactService) ListVersions(ctx context.Context, appName, userID, sessionID, filename string) ([]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.artifactPath(appName, userID, sessionID, filename)
	versions, exists := s.artifacts[path]
	if !exists || len(versions) == 0 {
		return []int{}, nil
	}

	result := make([]int, len(versions))
	for i := range versions {
		result[i] = i
	}

	return result, nil
}
