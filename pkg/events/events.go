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

package events

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/nvcnvn/adk-golang/pkg/models"
)

// Content is a convenience alias to models.Content to avoid importing models everywhere
type Content = models.Content

// Event represents an event in the agent system
type Event struct {
	// ID is a unique identifier for this event
	ID string `json:"id"`

	// InvocationID links this event to an invocation
	InvocationID string `json:"invocationId,omitempty"`

	// Author is the name of the agent that generated this event
	Author string `json:"author,omitempty"`

	// Branch identifies a specific branch in a conversation
	Branch string `json:"branch,omitempty"`

	// Content contains the actual content of this event
	Content *models.Content `json:"content,omitempty"`

	// Partial indicates if this is a partial response
	Partial bool `json:"partial,omitempty"`

	// ErrorCode holds an error code if the event represents an error
	ErrorCode string `json:"errorCode,omitempty"`

	// ErrorMessage holds an error message if the event represents an error
	ErrorMessage string `json:"errorMessage,omitempty"`

	// Interrupted indicates if the response was interrupted
	Interrupted bool `json:"interrupted,omitempty"`

	// LongRunningToolIDs contains IDs of long-running tools
	LongRunningToolIDs []string `json:"longRunningToolIds,omitempty"`

	// Actions contains actions associated with this event
	Actions *EventActions `json:"actions,omitempty"`
}

// NewEvent creates a new event with a unique ID
func NewEvent() *Event {
	return &Event{
		ID:      uuid.New().String(),
		Actions: NewEventActions(),
	}
}

// IsFinalResponse returns true if this event represents a final response
func (e *Event) IsFinalResponse() bool {
	// Final response if there's an error or a transfer to another agent
	if e.ErrorCode != "" || (e.Actions != nil && e.Actions.TransferToAgent != "") {
		return true
	}

	// Final response if there's content and it's not partial
	if e.Content != nil && !e.Partial {
		return true
	}

	return false
}

// GetFunctionCalls extracts function calls from the event content
func (e *Event) GetFunctionCalls() []*models.FunctionCall {
	functionCalls := make([]*models.FunctionCall, 0)

	if e.Content == nil {
		return functionCalls
	}

	for _, part := range e.Content.Parts {
		if part.FunctionCall != nil {
			functionCalls = append(functionCalls, part.FunctionCall)
		}
	}

	return functionCalls
}

// GenerateID generates a random ID for events.
func GenerateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 8

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// Initialize the random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}
