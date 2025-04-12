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

// Package sessions provides interfaces and implementations for session management.
package sessions

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nvcnvn/adk-golang/pkg/events"
)

// Session represents a conversation session.
type Session struct {
	// AppName is the name of the application
	AppName string `json:"appName"`

	// UserID is the ID of the user
	UserID string `json:"userId"`

	// ID is a unique identifier for this session
	ID string `json:"id"`

	// Events is a list of events in this session
	Events []*events.Event `json:"events"`

	// State contains session state data
	State *State `json:"-"` // Not directly serialized

	// StateMap is the serializable representation of State
	StateMap map[string]interface{} `json:"state"`

	// CreateTime is when the session was created
	CreateTime time.Time `json:"createTime"`

	// UpdateTime is when the session was last updated
	UpdateTime time.Time `json:"updateTime"`

	mu sync.RWMutex
}

// NewSession creates a new session with a unique ID.
func NewSession(appName, userID string, initialState map[string]interface{}, sessionID string) *Session {
	now := time.Now()

	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	if initialState == nil {
		initialState = make(map[string]interface{})
	}

	state := NewState(initialState, make(map[string]interface{}))

	return &Session{
		AppName:    appName,
		UserID:     userID,
		ID:         sessionID,
		Events:     []*events.Event{},
		State:      state,
		StateMap:   initialState,
		CreateTime: now,
		UpdateTime: now,
	}
}

// AddEvent adds an event to the session.
func (s *Session) AddEvent(event *events.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !event.Partial {
		s.updateSessionState(event)
	}

	s.Events = append(s.Events, event)
	s.UpdateTime = time.Now()
}

// GetEvent gets an event by ID.
func (s *Session) GetEvent(id string) *events.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, event := range s.Events {
		if event.ID == id {
			return event
		}
	}

	return nil
}

// GetState gets a value from the session state.
func (s *Session) GetState(key string) (interface{}, bool) {
	return s.State.Get(key)
}

// SetState sets a value in the session state.
func (s *Session) SetState(key string, value interface{}) {
	s.State.Set(key, value)
	s.mu.Lock()
	s.StateMap = s.State.ToMap()
	s.UpdateTime = time.Now()
	s.mu.Unlock()
}

// GetAllEvents returns all events in the session.
func (s *Session) GetAllEvents() []*events.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]*events.Event, len(s.Events))
	copy(events, s.Events)
	return events
}

// updateSessionState updates the session state based on an event's state delta
func (s *Session) updateSessionState(event *events.Event) {
	if event.Actions == nil || len(event.Actions.StateDelta) == 0 {
		return
	}

	// Filter out temporary state values
	stateDelta := make(map[string]interface{})
	for key, value := range event.Actions.StateDelta {
		if !strings.HasPrefix(key, TempPrefix) {
			stateDelta[key] = value
		}
	}

	if len(stateDelta) > 0 {
		s.State.Update(stateDelta)
		s.StateMap = s.State.ToMap()
	}
}
