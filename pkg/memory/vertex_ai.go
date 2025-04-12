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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/models"
	"github.com/nvcnvn/adk-golang/pkg/sessions"
	"github.com/nvcnvn/adk-golang/pkg/telemetry"
)

// VertexAiRagMemoryService is a memory service that uses Vertex AI RAG for storage and retrieval.
type VertexAiRagMemoryService struct {
	// RagCorpus is the name of the Vertex AI RAG corpus to use
	RagCorpus string

	// SimilarityTopK is the number of contexts to retrieve
	SimilarityTopK int

	// VectorDistanceThreshold only returns contexts with vector distance smaller than the threshold
	VectorDistanceThreshold float64

	mu sync.RWMutex
}

// RagContext represents a context returned by RAG retrieval queries
type RagContext struct {
	Text              string
	SourceDisplayName string
	VectorDistance    float64
}

// EventData represents an event in JSON format for storage
type EventData struct {
	Author    string `json:"author"`
	Timestamp int64  `json:"timestamp"`
	Text      string `json:"text"`
}

// NewVertexAiRagMemoryService creates a new VertexAiRagMemoryService.
func NewVertexAiRagMemoryService(ragCorpus string, similarityTopK int, vectorDistanceThreshold float64) *VertexAiRagMemoryService {
	return &VertexAiRagMemoryService{
		RagCorpus:               ragCorpus,
		SimilarityTopK:          similarityTopK,
		VectorDistanceThreshold: vectorDistanceThreshold,
	}
}

// AddSessionToMemory implements MemoryService.AddSessionToMemory.
func (s *VertexAiRagMemoryService) AddSessionToMemory(ctx context.Context, session *sessions.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a temporary file to store the session events
	tempFile, err := ioutil.TempFile("", "vertex_ai_rag_*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up the file when done

	// Process and write events to the temporary file
	outputLines := []string{}
	for _, event := range session.Events {
		if event.Content == nil || len(event.Content.Parts) == 0 {
			continue
		}

		// Extract text from event parts and join them
		var textParts []string
		for _, part := range event.Content.Parts {
			if part.Text != "" {
				// Replace newlines with spaces to keep it as a single line
				text := strings.ReplaceAll(part.Text, "\n", " ")
				textParts = append(textParts, text)
			}
		}

		if len(textParts) > 0 {
			// Create event data and convert to JSON
			eventData := EventData{
				Author:    event.Author,
				Timestamp: time.Now().Unix(), // Using current time since Go Event doesn't have timestamp
				Text:      strings.Join(textParts, ". "),
			}

			jsonData, err := json.Marshal(eventData)
			if err != nil {
				telemetry.Warning("Failed to marshal event data: %v", err)
				continue
			}

			outputLines = append(outputLines, string(jsonData))
		}
	}

	// Write the output to the temporary file
	output := strings.Join(outputLines, "\n")
	if _, err := tempFile.Write([]byte(output)); err != nil {
		return fmt.Errorf("failed to write to temporary file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %v", err)
	}

	// Upload the file to Vertex AI RAG
	displayName := fmt.Sprintf("%s.%s.%s", session.AppName, session.UserID, session.ID)
	if err := s.uploadFileToRag(ctx, tempFile.Name(), displayName); err != nil {
		return fmt.Errorf("failed to upload file to RAG: %v", err)
	}

	return nil
}

// SearchMemory implements MemoryService.SearchMemory.
func (s *VertexAiRagMemoryService) SearchMemory(ctx context.Context, appName, userID, query string) (*SearchMemoryResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := &SearchMemoryResponse{
		Memories: []*MemoryResult{},
	}

	// Use Vertex AI RAG for retrieval
	contexts, err := s.retrievalQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query RAG: %v", err)
	}

	// Process results and organize by session
	sessionEventsMap := make(map[string][]*EventList)
	for _, context := range contexts {
		// Extract session ID from display name
		parts := strings.Split(context.SourceDisplayName, ".")
		if len(parts) != 3 {
			continue
		}

		// Optional filtering by app_name and user_id
		contextAppName := parts[0]
		contextUserID := parts[1]
		sessionID := parts[2]

		// Filter by app_name and user_id if specified
		if appName != "" && contextAppName != appName {
			continue
		}
		if userID != "" && contextUserID != userID {
			continue
		}

		// Process the text content to extract events
		eventsWithTimestamp, err := s.parseContextToEvents(context.Text)
		if err != nil {
			telemetry.Warning("Failed to parse context to events: %v", err)
			continue
		}

		if len(eventsWithTimestamp) > 0 {
			sessionEventsMap[sessionID] = append(sessionEventsMap[sessionID], &EventList{
				Events: eventsWithTimestamp,
			})
		}
	}

	// Merge events from the same session
	for sessionID, eventLists := range sessionEventsMap {
		mergedEventLists := s.mergeEventLists(eventLists)

		// Add each merged list as a memory result
		for _, eventList := range mergedEventLists {
			// Sort events by ID for consistency (since we don't have timestamps)
			sort.Slice(eventList.Events, func(i, j int) bool {
				return eventList.Events[i].ID < eventList.Events[j].ID
			})

			response.Memories = append(response.Memories, &MemoryResult{
				SessionID: sessionID,
				Events:    eventList.Events,
			})
		}
	}

	return response, nil
}

// EventList represents a list of events that will be merged if they overlap.
type EventList struct {
	Events []*events.Event
	// We'll use the Event.ID field to track uniqueness
}

// Helper functions

// uploadFileToRag uploads a file to the Vertex AI RAG corpus.
func (s *VertexAiRagMemoryService) uploadFileToRag(ctx context.Context, filePath, displayName string) error {
	// This is a simplified version as the actual implementation would require
	// using the Vertex AI client libraries to create or upload files to a RAG corpus
	telemetry.Info("Uploading file %s to RAG corpus %s with display name %s", filePath, s.RagCorpus, displayName)

	// TODO: Implement actual upload to Vertex AI RAG when the Go client libraries are available
	// For now, we'll just simulate success
	return nil
}

// retrievalQuery performs a retrieval query against the RAG corpus.
func (s *VertexAiRagMemoryService) retrievalQuery(ctx context.Context, query string) ([]*RagContext, error) {
	// This is a simplified version that would normally use the Vertex AI client libraries
	telemetry.Info("Querying RAG corpus %s for: %s", s.RagCorpus, query)

	// TODO: Implement actual retrieval from Vertex AI RAG when the Go client libraries are available
	// For now, return an empty slice
	return []*RagContext{}, nil
}

// parseContextToEvents parses the text content from a RAG context into events.
func (s *VertexAiRagMemoryService) parseContextToEvents(text string) ([]*events.Event, error) {
	if text == "" {
		return []*events.Event{}, nil
	}

	lines := strings.Split(text, "\n")
	eventsResult := []*events.Event{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to parse as JSON
		var eventData EventData
		if err := json.Unmarshal([]byte(line), &eventData); err != nil {
			telemetry.Warning("Failed to parse line as JSON: %s", line)
			continue
		}

		// Create an event from the parsed data
		event := events.NewEvent()
		event.Author = eventData.Author
		event.Content = &models.Content{
			Parts: []*models.Part{
				{
					Text: eventData.Text,
				},
			},
		}

		eventsResult = append(eventsResult, event)
	}

	return eventsResult, nil
}

// mergeEventLists merges event lists that have overlapping IDs.
func (s *VertexAiRagMemoryService) mergeEventLists(eventLists []*EventList) []*EventList {
	if len(eventLists) <= 1 {
		return eventLists
	}

	merged := []*EventList{}

	// Create a copy of the input lists
	remaining := make([]*EventList, len(eventLists))
	copy(remaining, eventLists)

	for len(remaining) > 0 {
		// Take the first list
		current := remaining[0]
		remaining = remaining[1:]

		// Create a set of IDs in the current list
		currentIDs := make(map[string]struct{})
		for _, event := range current.Events {
			currentIDs[event.ID] = struct{}{}
		}

		mergeFound := true

		// Keep merging until no new overlap is found
		for mergeFound {
			mergeFound = false
			newRemaining := []*EventList{}

			for _, other := range remaining {
				hasOverlap := false

				// Check if there's any overlap in IDs
				for _, event := range other.Events {
					if _, exists := currentIDs[event.ID]; exists {
						hasOverlap = true
						break
					}
				}

				if hasOverlap {
					// Merge with current list
					for _, event := range other.Events {
						if _, exists := currentIDs[event.ID]; !exists {
							current.Events = append(current.Events, event)
							currentIDs[event.ID] = struct{}{}
						}
					}
					mergeFound = true
				} else {
					// Keep this list for later
					newRemaining = append(newRemaining, other)
				}
			}

			remaining = newRemaining
		}

		merged = append(merged, current)
	}

	return merged
}
