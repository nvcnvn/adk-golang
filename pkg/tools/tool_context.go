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

package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/models"
)

// InvocationContext is a forward declaration to avoid import cycle
type InvocationContext interface {
	GetID() string
	GetAgentName() string
	IsEndInvocation() bool
	SetEndInvocation(end bool)
	GetTranscriptionCache() interface{}
}

// ToolContext provides context for tool execution
type ToolContext struct {
	// InvocationContext is the parent invocation context
	InvocationContext InvocationContext

	// EventActions contains actions associated with an event
	EventActions *events.EventActions
}

// LlmToolAdaptor wraps an existing Tool to add LLM-specific functionality
// This is different from LlmToolWrapper - it's an adapter that actually
// implements the Tool interface by delegating to the wrapped tool
type LlmToolAdaptor struct {
	// The base tool
	tool Tool

	// Whether this tool takes a long time to execute
	isLongRunning bool

	// ProcessLlmRequestFunc is called before the LLM is called
	processLlmRequestFunc func(ctx context.Context, toolContext *ToolContext, llmRequest *models.LlmRequest) error
}

// NewLlmToolAdaptor creates a new LlmToolAdaptor from a Tool
func NewLlmToolAdaptor(tool Tool, isLongRunning bool) *LlmToolAdaptor {
	return &LlmToolAdaptor{
		tool:          tool,
		isLongRunning: isLongRunning,
	}
}

// Name returns the name of the tool
func (a *LlmToolAdaptor) Name() string {
	return a.tool.Name()
}

// Description returns the description of the tool
func (a *LlmToolAdaptor) Description() string {
	return a.tool.Description()
}

// Execute executes the tool with the given input
func (a *LlmToolAdaptor) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return a.tool.Execute(ctx, input)
}

// Schema returns the schema for the tool
func (a *LlmToolAdaptor) Schema() ToolSchema {
	return a.tool.Schema()
}

// IsLongRunning returns whether this tool takes a long time to execute
func (a *LlmToolAdaptor) IsLongRunning() bool {
	return a.isLongRunning
}

// ProcessLlmRequest processes the LLM request before it is sent
func (a *LlmToolAdaptor) ProcessLlmRequest(ctx context.Context, toolContext *ToolContext, llmRequest *models.LlmRequest) error {
	if a.processLlmRequestFunc != nil {
		return a.processLlmRequestFunc(ctx, toolContext, llmRequest)
	}
	return nil
}

// ExecuteFunctionCall executes a function call using the wrapped tool
func (a *LlmToolAdaptor) ExecuteFunctionCall(ctx context.Context, toolContext *ToolContext, functionCall *models.FunctionCall) (string, error) {
	// Parse the arguments from the function call
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(functionCall.Arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse function arguments: %v", err)
	}

	// Execute the wrapped tool
	result, err := a.tool.Execute(ctx, args)
	if err != nil {
		return "", err
	}

	// Convert the result to JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %v", err)
	}

	return string(resultJSON), nil
}

// SetProcessLlmRequestFunc sets the function that processes LLM requests
func (a *LlmToolAdaptor) SetProcessLlmRequestFunc(fn func(ctx context.Context, toolContext *ToolContext, llmRequest *models.LlmRequest) error) {
	a.processLlmRequestFunc = fn
}
