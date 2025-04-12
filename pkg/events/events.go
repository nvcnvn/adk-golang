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

// Package events provides types and functionality for events in the Agent Development Kit.
package events

import (
	"math/rand"
	"time"
)

// Content represents a message part in an event.
type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

// Part represents a part of content in an event.
type Part struct {
	Text                string               `json:"text,omitempty"`
	FunctionCall        *FunctionCall        `json:"function_call,omitempty"`
	FunctionResponse    *FunctionResponse    `json:"function_response,omitempty"`
	CodeExecutionResult *CodeExecutionResult `json:"code_execution_result,omitempty"`
}

// FunctionCall represents a function call in an event.
type FunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
	ID   string                 `json:"id,omitempty"`
}

// FunctionResponse represents a response to a function call.
type FunctionResponse struct {
	Name     string      `json:"name"`
	Response interface{} `json:"response"`
	ID       string      `json:"id,omitempty"`
	Status   string      `json:"status,omitempty"`
}

// CodeExecutionResult represents the result of code execution.
type CodeExecutionResult struct {
	Output string `json:"output"`
	Status string `json:"status"`
}

// Event represents an event in a conversation between agents and users.
// It stores content of the conversation and actions taken by agents.
type Event struct {
	ID           string        `json:"id"`
	InvocationID string        `json:"invocation_id"`
	Author       string        `json:"author"`
	Actions      *EventActions `json:"actions,omitempty"`
	Content      *Content      `json:"content,omitempty"`
	Timestamp    float64       `json:"timestamp"`

	// LongRunningToolIDs is the set of IDs of long running function calls.
	// Agent client will know from this field about which function call is long running.
	// Only valid for function call events.
	LongRunningToolIDs map[string]struct{} `json:"long_running_tool_ids,omitempty"`

	// Branch is used to track the agent hierarchy path, e.g.: agent_1.agent_2.agent_3
	// where agent_1 is the parent of agent_2, and agent_2 is the parent of agent_3.
	// This is used when multiple sub-agents shouldn't see their peer agents' conversation history.
	Branch string `json:"branch,omitempty"`

	// Additional error fields
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`

	// Indicates whether the event is partial and more updates will follow
	Partial bool `json:"partial,omitempty"`
	// Indicates whether this event completes the turn
	TurnComplete bool `json:"turn_complete,omitempty"`
	// Indicates whether the processing was interrupted
	Interrupted bool `json:"interrupted,omitempty"`
}

// NewEvent creates a new event with default values.
func NewEvent(author string) *Event {
	now := time.Now()
	return &Event{
		ID:        GenerateID(),
		Author:    author,
		Timestamp: float64(now.UnixNano()) / 1e9,
		Actions:   NewEventActions(),
	}
}

// IsFinalResponse determines if this event is a final response from the agent.
func (e *Event) IsFinalResponse() bool {
	if e.Actions.SkipSummarization || len(e.LongRunningToolIDs) > 0 {
		return true
	}

	return len(e.GetFunctionCalls()) == 0 &&
		len(e.GetFunctionResponses()) == 0 &&
		!e.Partial &&
		!e.HasTrailingCodeExecutionResult()
}

// GetFunctionCalls returns all function calls in the event.
func (e *Event) GetFunctionCalls() []*FunctionCall {
	var calls []*FunctionCall
	if e.Content != nil && len(e.Content.Parts) > 0 {
		for _, part := range e.Content.Parts {
			if part.FunctionCall != nil {
				calls = append(calls, part.FunctionCall)
			}
		}
	}
	return calls
}

// GetFunctionResponses returns all function responses in the event.
func (e *Event) GetFunctionResponses() []*FunctionResponse {
	var responses []*FunctionResponse
	if e.Content != nil && len(e.Content.Parts) > 0 {
		for _, part := range e.Content.Parts {
			if part.FunctionResponse != nil {
				responses = append(responses, part.FunctionResponse)
			}
		}
	}
	return responses
}

// HasTrailingCodeExecutionResult returns whether the event has a trailing code execution result.
func (e *Event) HasTrailingCodeExecutionResult() bool {
	if e.Content != nil && len(e.Content.Parts) > 0 {
		lastPart := e.Content.Parts[len(e.Content.Parts)-1]
		return lastPart.CodeExecutionResult != nil
	}
	return false
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
