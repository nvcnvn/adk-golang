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

// Package artifacts provides functionality for storing and retrieving artifacts.
package artifacts

import (
	"context"
)

// Part represents an artifact part, similar to google.genai.types.Part in Python.
type Part struct {
	// Text contains the textual content of the artifact.
	// Either Text or Data should be populated, but not both.
	Text string

	// Data contains the binary content of the artifact.
	// Either Text or Data should be populated, but not both.
	Data []byte

	// MimeType represents the MIME type of the content.
	MimeType string
}

// FromText creates a Part instance from text content.
func FromText(text string, mimeType string) Part {
	return Part{
		Text:     text,
		MimeType: mimeType,
	}
}

// FromBytes creates a Part instance from binary data.
func FromBytes(data []byte, mimeType string) Part {
	return Part{
		Data:     data,
		MimeType: mimeType,
	}
}

// ArtifactService defines the interface for artifact services.
type ArtifactService interface {
	// SaveArtifact saves an artifact to the artifact service storage.
	// The artifact is identified by app name, user ID, session ID, and filename.
	// After saving, it returns a revision ID to identify the artifact version.
	SaveArtifact(ctx context.Context, appName, userID, sessionID, filename string, artifact Part) (int, error)

	// LoadArtifact gets an artifact from the artifact service storage.
	// If version is nil, the latest version will be returned.
	LoadArtifact(ctx context.Context, appName, userID, sessionID, filename string, version *int) (*Part, error)

	// ListArtifactKeys lists all the artifact filenames within a session.
	ListArtifactKeys(ctx context.Context, appName, userID, sessionID string) ([]string, error)

	// DeleteArtifact deletes an artifact.
	DeleteArtifact(ctx context.Context, appName, userID, sessionID, filename string) error

	// ListVersions lists all versions of an artifact.
	ListVersions(ctx context.Context, appName, userID, sessionID, filename string) ([]int, error)
}
