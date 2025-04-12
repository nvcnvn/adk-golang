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

package memory

import (
	"context"
	"strings"
	"sync"

	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/sessions"
)

// InMemoryMemoryService is an in-memory memory service for prototyping purposes only.
// It uses keyword matching instead of semantic search.
type InMemoryMemoryService struct {
	// sessionEvents maps app_name/user_id/session_id to a list of events
	sessionEvents map[string][]*events.Event
	mu            sync.RWMutex
}

// NewInMemoryMemoryService creates a new InMemoryMemoryService.
func NewInMemoryMemoryService() *InMemoryMemoryService {
	return &InMemoryMemoryService{
		sessionEvents: make(map[string][]*events.Event),
	}
}

// AddSessionToMemory implements MemoryService.AddSessionToMemory.
func (s *InMemoryMemoryService) AddSessionToMemory(ctx context.Context, session *sessions.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := formatSessionKey(session.AppName, session.UserID, session.ID)

	// Filter out events with no content
	var eventsWithContent []*events.Event
	for _, event := range session.Events {
		if event.Content != nil && len(event.Content.Parts) > 0 {
			eventsWithContent = append(eventsWithContent, event)
		}
	}

	s.sessionEvents[key] = eventsWithContent
	return nil
}

// SearchMemory implements MemoryService.SearchMemory.
func (s *InMemoryMemoryService) SearchMemory(ctx context.Context, appName, userID, query string) (*SearchMemoryResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keywords := tokenize(query)
	response := &SearchMemoryResponse{
		Memories: []*MemoryResult{},
	}

	prefix := formatSessionKeyPrefix(appName, userID)

	for key, sessionEvents := range s.sessionEvents {
		// Filter sessions by app_name and user_id
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		var matchedEvents []*events.Event
		for _, event := range sessionEvents {
			if event.Content == nil || len(event.Content.Parts) == 0 {
				continue
			}

			// Create a concatenated text from all parts
			var text strings.Builder
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					if text.Len() > 0 {
						text.WriteString("\n")
					}
					text.WriteString(part.Text)
				}
			}

			// Check if any keyword is in the text
			if containsAny(strings.ToLower(text.String()), keywords) {
				matchedEvents = append(matchedEvents, event)
			}
		}

		if len(matchedEvents) > 0 {
			sessionID := extractSessionID(key)
			response.Memories = append(response.Memories, &MemoryResult{
				SessionID: sessionID,
				Events:    matchedEvents,
			})
		}
	}

	return response, nil
}

// Helper functions

// formatSessionKey formats a key for the sessionEvents map.
func formatSessionKey(appName, userID, sessionID string) string {
	return appName + "/" + userID + "/" + sessionID
}

// formatSessionKeyPrefix formats a prefix for filtering keys by app_name and user_id.
func formatSessionKeyPrefix(appName, userID string) string {
	return appName + "/" + userID + "/"
}

// extractSessionID extracts the session ID from a key.
func extractSessionID(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[2]
}

// tokenize splits a query into lowercase keywords.
func tokenize(query string) []string {
	return strings.Fields(strings.ToLower(query))
}

// containsAny checks if text contains any of the keywords.
func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}
