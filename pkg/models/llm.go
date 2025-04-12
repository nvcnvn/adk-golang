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

package models

import (
	"context"
)

// LlmRequest represents a request to an LLM model
type LlmRequest struct {
	// Contents contains the conversation history to be sent to the model
	Contents *Content `json:"contents,omitempty"`

	// Tools is the list of tools available to the model
	Tools []*Tool `json:"tools,omitempty"`

	// ToolsDict is a map of tool name to tool for faster lookup
	ToolsDict map[string]*Tool `json:"-"`

	// System instructions to send to the model
	SystemInstructions string `json:"systemInstructions,omitempty"`

	// Temperature controls randomness in the output
	Temperature float64 `json:"temperature,omitempty"`

	// TopP controls diversity in the output
	TopP float64 `json:"topP,omitempty"`

	// TopK controls the number of tokens to consider
	TopK int `json:"topK,omitempty"`

	// MaxTokens limits the maximum number of tokens in the response
	MaxTokens int `json:"maxTokens,omitempty"`

	// CandidateCount specifies the number of response candidates to generate
	CandidateCount int `json:"candidateCount,omitempty"`
}

// LlmResponse represents a response from an LLM model
type LlmResponse struct {
	// Content contains the response content from the model
	Content *Content `json:"content,omitempty"`

	// Partial indicates if this is a partial response in streaming mode
	Partial bool `json:"partial,omitempty"`

	// ErrorCode holds an error code if the model call failed
	ErrorCode string `json:"errorCode,omitempty"`

	// ErrorMessage holds an error message if the model call failed
	ErrorMessage string `json:"errorMessage,omitempty"`

	// Interrupted indicates if the response was interrupted
	Interrupted bool `json:"interrupted,omitempty"`

	// TurnComplete indicates if the turn is complete (used in live mode)
	TurnComplete bool `json:"turnComplete,omitempty"`
}

// Content represents the content in a message, containing one or more parts
type Content struct {
	// Parts contains the individual content parts
	Parts []*Part `json:"parts,omitempty"`
}

// GetText returns the text from the first part with text
func (c *Content) GetText() string {
	if c == nil || len(c.Parts) == 0 {
		return ""
	}

	for _, part := range c.Parts {
		if part.Text != "" {
			return part.Text
		}
	}

	return ""
}

// Part represents a part of content, which can be text, function call, function response, etc.
type Part struct {
	// Text is plain text content
	Text string `json:"text,omitempty"`

	// Role is the role of the part (system, user, assistant)
	Role string `json:"role,omitempty"`

	// FunctionCall represents a function call from the model
	FunctionCall *FunctionCall `json:"functionCall,omitempty"`

	// FunctionResponse represents a response to a function call
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`

	// AuthRequest represents an authentication request
	AuthRequest *AuthRequest `json:"authRequest,omitempty"`
}

// FunctionCall represents a call to a function
type FunctionCall struct {
	// Name is the name of the function to call
	Name string `json:"name"`

	// Arguments are the arguments to the function
	Arguments string `json:"arguments,omitempty"`

	// ID is a unique identifier for this function call
	ID string `json:"id,omitempty"`
}

// FunctionResponse represents a response from a function call
type FunctionResponse struct {
	// Name is the name of the function that was called
	Name string `json:"name"`

	// Content is the result of the function call
	Content string `json:"content,omitempty"`

	// ID is the identifier matching the function call
	ID string `json:"id,omitempty"`

	// AuthRequest is an optional auth request if authentication is needed
	AuthRequest *AuthRequest `json:"authRequest,omitempty"`
}

// AuthRequest represents a request for authentication
type AuthRequest struct {
	// Provider is the authentication provider
	Provider string `json:"provider"`

	// Scope is the requested authentication scope
	Scope string `json:"scope,omitempty"`

	// RedirectURI is the URI to redirect to after authentication
	RedirectURI string `json:"redirectUri,omitempty"`
}

// Tool represents a tool that can be used by the model
type Tool struct {
	// Name is the name of the tool
	Name string `json:"name"`

	// Description is a description of what the tool does
	Description string `json:"description,omitempty"`

	// InputSchema defines the expected input schema
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`

	// IsLongRunning indicates if this tool takes a long time to execute
	IsLongRunning bool `json:"isLongRunning,omitempty"`
}

// BaseLlm defines the interface for an LLM model
type BaseLlm interface {
	// GenerateContent generates content from the model based on the request
	GenerateContent(ctx context.Context, request *LlmRequest, stream bool) (<-chan *LlmResponse, error)

	// Connect establishes a connection to the model for live interactions
	Connect(request *LlmRequest) (BaseLlmConnection, error)
}

// BaseLlmConnection represents a live connection to an LLM
type BaseLlmConnection interface {
	// SendHistory sends conversation history to the model
	SendHistory(content *Content) error

	// SendContent sends content to the model during a conversation
	SendContent(content *Content) error

	// SendRealtime sends real-time data to the model
	SendRealtime(data []byte) error

	// Receive receives responses from the model
	Receive() (<-chan *LlmResponse, error)

	// Close closes the connection
	Close() error
}
