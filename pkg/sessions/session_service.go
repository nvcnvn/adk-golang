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

package sessions

import (
	"context"

	"github.com/nvcnvn/adk-golang/pkg/events"
)

// GetSessionConfig contains options for retrieving a session
type GetSessionConfig struct {
	// NumRecentEvents limits the number of events to return, if set
	NumRecentEvents int
	// AfterTimestamp filters events after the specified timestamp
	AfterTimestamp float64
}

// ListSessionsResponse contains the result of a list sessions operation
type ListSessionsResponse struct {
	Sessions []*Session
}

// ListEventsResponse contains the result of a list events operation
type ListEventsResponse struct {
	Events        []*events.Event
	NextPageToken string
}

// SessionService defines the interface for session management
type SessionService interface {
	// CreateSession creates a new session
	CreateSession(ctx context.Context, appName, userID string, state map[string]interface{}, sessionID string) (*Session, error)

	// GetSession retrieves a session by ID
	GetSession(ctx context.Context, appName, userID, sessionID string, config *GetSessionConfig) (*Session, error)

	// ListSessions lists all sessions for a user
	ListSessions(ctx context.Context, appName, userID string) (*ListSessionsResponse, error)

	// DeleteSession deletes a session
	DeleteSession(ctx context.Context, appName, userID, sessionID string) error

	// ListEvents lists events in a session
	ListEvents(ctx context.Context, appName, userID, sessionID string) (*ListEventsResponse, error)

	// CloseSession finalizes and closes a session
	CloseSession(ctx context.Context, session *Session) error

	// AppendEvent adds an event to a session
	AppendEvent(ctx context.Context, session *Session, event *events.Event) (*events.Event, error)
}
