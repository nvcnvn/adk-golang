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
	"fmt"
	"sync"

	"github.com/nvcnvn/adk-golang/pkg/events"
)

// InMemorySessionService provides an in-memory implementation of SessionService
type InMemorySessionService struct {
	// Sessions are stored in a nested map structure:
	// app_name -> user_id -> session_id -> Session
	sessions map[string]map[string]map[string]*Session
	mu       sync.RWMutex
}

// NewInMemorySessionService creates a new in-memory session service
func NewInMemorySessionService() *InMemorySessionService {
	return &InMemorySessionService{
		sessions: make(map[string]map[string]map[string]*Session),
	}
}

// CreateSession creates a new session in memory
func (s *InMemorySessionService) CreateSession(
	ctx context.Context,
	appName, userID string,
	state map[string]interface{},
	sessionID string,
) (*Session, error) {
	session := NewSession(appName, userID, state, sessionID)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure nested maps exist
	if _, exists := s.sessions[appName]; !exists {
		s.sessions[appName] = make(map[string]map[string]*Session)
	}
	if _, exists := s.sessions[appName][userID]; !exists {
		s.sessions[appName][userID] = make(map[string]*Session)
	}

	// Store the session
	s.sessions[appName][userID][session.ID] = session
	return session, nil
}

// GetSession retrieves a session by ID
func (s *InMemorySessionService) GetSession(
	ctx context.Context,
	appName, userID, sessionID string,
	config *GetSessionConfig,
) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, err := s.findSession(appName, userID, sessionID)
	if err != nil {
		return nil, err
	}

	// Apply config if provided
	if config != nil {
		return s.applySessionConfig(session, config), nil
	}

	return session, nil
}

// ListSessions lists all sessions for a user
func (s *InMemorySessionService) ListSessions(
	ctx context.Context,
	appName, userID string,
) (*ListSessionsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := &ListSessionsResponse{
		Sessions: make([]*Session, 0),
	}

	// Check if app and user exist
	if _, exists := s.sessions[appName]; !exists {
		return response, nil
	}
	if _, exists := s.sessions[appName][userID]; !exists {
		return response, nil
	}

	// Copy sessions without events and state to reduce response size
	for _, session := range s.sessions[appName][userID] {
		// Create a shallow copy of the session without events
		sessionCopy := &Session{
			ID:         session.ID,
			AppName:    session.AppName,
			UserID:     session.UserID,
			StateMap:   session.StateMap,
			CreateTime: session.CreateTime,
			UpdateTime: session.UpdateTime,
		}
		response.Sessions = append(response.Sessions, sessionCopy)
	}

	return response, nil
}

// DeleteSession deletes a session
func (s *InMemorySessionService) DeleteSession(
	ctx context.Context,
	appName, userID, sessionID string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if session exists before attempting deletion
	if _, err := s.findSession(appName, userID, sessionID); err != nil {
		return err
	}

	delete(s.sessions[appName][userID], sessionID)
	return nil
}

// ListEvents lists events in a session
func (s *InMemorySessionService) ListEvents(
	ctx context.Context,
	appName, userID, sessionID string,
) (*ListEventsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, err := s.findSession(appName, userID, sessionID)
	if err != nil {
		return nil, err
	}

	// Create a copy of events to avoid race conditions
	events := make([]*events.Event, len(session.Events))
	copy(events, session.Events)

	return &ListEventsResponse{
		Events:        events,
		NextPageToken: "", // In-memory implementation doesn't use pagination
	}, nil
}

// CloseSession finalizes a session (no-op in memory implementation)
func (s *InMemorySessionService) CloseSession(ctx context.Context, session *Session) error {
	// No specific action needed for in-memory sessions
	return nil
}

// AppendEvent adds an event to a session
func (s *InMemorySessionService) AppendEvent(
	ctx context.Context,
	session *Session,
	event *events.Event,
) (*events.Event, error) {
	if event.Partial {
		return event, nil
	}

	// Update session with the new event
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the actual stored session to update
	storedSession, err := s.findSession(session.AppName, session.UserID, session.ID)
	if err != nil {
		return nil, err
	}

	// Add the event
	storedSession.AddEvent(event)

	// Update the original session to match the stored one
	session.Events = make([]*events.Event, len(storedSession.Events))
	copy(session.Events, storedSession.Events)
	session.StateMap = storedSession.StateMap
	session.UpdateTime = storedSession.UpdateTime

	return event, nil
}

// findSession is a helper method to find a session
// Caller must hold at least a read lock
func (s *InMemorySessionService) findSession(
	appName, userID, sessionID string,
) (*Session, error) {
	// Check if app exists
	appSessions, appExists := s.sessions[appName]
	if !appExists {
		return nil, fmt.Errorf("app %s not found", appName)
	}

	// Check if user exists
	userSessions, userExists := appSessions[userID]
	if !userExists {
		return nil, fmt.Errorf("user %s not found for app %s", userID, appName)
	}

	// Check if session exists
	session, sessionExists := userSessions[sessionID]
	if !sessionExists {
		return nil, fmt.Errorf("session %s not found for user %s in app %s", sessionID, userID, appName)
	}

	return session, nil
}

// applySessionConfig applies configuration options to a session copy
func (s *InMemorySessionService) applySessionConfig(session *Session, config *GetSessionConfig) *Session {
	// Create a copy to avoid modifying the original
	sessionCopy := &Session{
		ID:         session.ID,
		AppName:    session.AppName,
		UserID:     session.UserID,
		State:      session.State,
		StateMap:   session.StateMap,
		CreateTime: session.CreateTime,
		UpdateTime: session.UpdateTime,
	}

	// Filter events based on timestamp if specified
	if config.AfterTimestamp > 0 {
		// In a real implementation, we would filter events based on timestamp
		// For simplicity in the in-memory implementation, we'll include all events
		filteredEvents := make([]*events.Event, 0)

		for _, event := range session.Events {
			filteredEvents = append(filteredEvents, event)
		}

		sessionCopy.Events = filteredEvents
	} else {
		// Otherwise copy all events
		sessionCopy.Events = make([]*events.Event, len(session.Events))
		copy(sessionCopy.Events, session.Events)
	}

	// Limit the number of events if specified
	if config.NumRecentEvents > 0 && len(sessionCopy.Events) > config.NumRecentEvents {
		start := len(sessionCopy.Events) - config.NumRecentEvents
		sessionCopy.Events = sessionCopy.Events[start:]
	}

	return sessionCopy
}
