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

// Package memory provides interfaces and implementations for memory services.
package memory

import (
	"context"

	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/sessions"
)

// MemoryResult represents a single memory retrieval result.
type MemoryResult struct {
	// SessionID is the session ID associated with the memory
	SessionID string `json:"sessionId"`

	// Events is a list of events in the session
	Events []*events.Event `json:"events"`
}

// SearchMemoryResponse represents the response from a memory search.
type SearchMemoryResponse struct {
	// Memories is a list of memory results matching the search query
	Memories []*MemoryResult `json:"memories"`
}

// MemoryService defines the interface for memory services.
// The service provides functionalities to ingest sessions into memory
// so that the memory can be used for user queries.
type MemoryService interface {
	// AddSessionToMemory adds a session to the memory service.
	// A session may be added multiple times during its lifetime.
	AddSessionToMemory(ctx context.Context, session *sessions.Session) error

	// SearchMemory searches for sessions that match the query.
	SearchMemory(ctx context.Context, appName, userID, query string) (*SearchMemoryResponse, error)
}
