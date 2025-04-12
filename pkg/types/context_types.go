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

// Package types provides shared type definitions used across the ADK
package types

import (
	"fmt"
	"sync"
)

// StreamingMode defines how responses should be streamed
type StreamingMode string

const (
	// StreamingModeNone means no streaming
	StreamingModeNone StreamingMode = "none"

	// StreamingModeSSE means server-sent events streaming
	StreamingModeSSE StreamingMode = "sse"
)

// RunConfig holds configuration for an agent invocation
type RunConfig struct {
	// StreamingMode controls how responses are streamed
	StreamingMode StreamingMode `json:"streamingMode,omitempty"`

	// MaxLlmCalls limits the number of LLM calls
	MaxLlmCalls int `json:"maxLlmCalls,omitempty"`

	// SupportCFC indicates if client-function-call (CFC) is supported
	SupportCFC bool `json:"supportCfc,omitempty"`
}

// TranscriptionEntry represents an audio transcription entry
type TranscriptionEntry struct {
	// Role is the role of the transcription (user or model)
	Role string `json:"role"`

	// Data contains the transcribed content
	Data interface{} `json:"data"`
}

// InvocationContextData holds the data shared between packages that's needed in InvocationContext
type InvocationContextData struct {
	// InvocationID is a unique identifier for this invocation
	InvocationID string `json:"invocationId"`

	// RunConfig contains configuration for this invocation
	RunConfig *RunConfig `json:"runConfig,omitempty"`

	// EndInvocation indicates if the invocation should end
	EndInvocation bool `json:"endInvocation,omitempty"`

	// Branch is an optional branch identifier
	Branch string `json:"branch,omitempty"`

	// TranscriptionCache holds cached transcriptions
	TranscriptionCache []TranscriptionEntry `json:"-"`

	// LlmCallCount counts the number of LLM calls made
	llmCallCount int

	// mu protects concurrent access
	mu sync.Mutex
}

// IncrementLlmCallCount increments and checks the LLM call count
func (ctx *InvocationContextData) IncrementLlmCallCount() error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.llmCallCount++

	if ctx.RunConfig.MaxLlmCalls > 0 && ctx.llmCallCount > ctx.RunConfig.MaxLlmCalls {
		return fmt.Errorf("maximum number of LLM calls (%d) exceeded", ctx.RunConfig.MaxLlmCalls)
	}

	return nil
}

// GetLlmCallCount returns the current LLM call count
func (ctx *InvocationContextData) GetLlmCallCount() int {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	return ctx.llmCallCount
}

// EventActionsData contains event actions that can be shared across packages
type EventActionsData struct {
	// TransferToAgent indicates which agent to transfer control to
	TransferToAgent string `json:"transferToAgent,omitempty"`
}
