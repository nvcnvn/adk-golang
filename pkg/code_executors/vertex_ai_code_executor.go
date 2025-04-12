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

// Package code_executors provides functionality for executing code snippets.
package code_executors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/telemetry"
)

// VertexAICodeExecutor uses Vertex AI's code execution capabilities.
type VertexAICodeExecutor struct {
	config CodeExecConfig
	// Add Vertex AI client and configuration here
}

// VertexAICodeExecutorOption is a functional option for VertexAICodeExecutor
type VertexAICodeExecutorOption func(*VertexAICodeExecutor)

// WithVertexAICodeExecConfig sets the config for the Vertex AI code executor
func WithVertexAICodeExecConfig(config CodeExecConfig) VertexAICodeExecutorOption {
	return func(e *VertexAICodeExecutor) {
		e.config = config
	}
}

// NewVertexAICodeExecutor creates a new VertexAICodeExecutor
func NewVertexAICodeExecutor(opts ...VertexAICodeExecutorOption) (*VertexAICodeExecutor, error) {
	executor := &VertexAICodeExecutor{
		config: DefaultCodeExecConfig(),
	}

	// Apply options
	for _, opt := range opts {
		opt(executor)
	}

	// Retrieve and validate Vertex AI credentials
	// In a real implementation, this would connect to Vertex AI

	return executor, nil
}

// ExecuteCode executes code using Vertex AI and returns the result
func (e *VertexAICodeExecutor) ExecuteCode(
	invocationContext *InvocationContext,
	input *CodeExecutionInput,
) (*ExecutionResult, error) {
	// Create a span for tracking this execution
	ctx, span := telemetry.StartSpan(invocationContext.Context, "VertexAICodeExecutor.ExecuteCode")
	defer span.End()

	// Publish event before execution
	if invocationContext.Events != nil {
		invocationContext.Events.Publish(ctx, events.ToolCalled, map[string]interface{}{
			"tool": "vertex_ai_code_executor",
			"code": input.Code,
		})
	}

	// In a real implementation, this would:
	// 1. Send the code to Vertex AI for execution
	// 2. Wait for and retrieve the results

	// For now, provide a mock implementation
	if strings.Contains(input.Code, "error") {
		err := errors.New("error executing code in Vertex AI")
		span.SetAttribute("error", err.Error())
		if invocationContext.Events != nil {
			invocationContext.Events.Publish(ctx, events.ToolError, map[string]interface{}{
				"tool":  "vertex_ai_code_executor",
				"error": err.Error(),
			})
		}
		return nil, err
	}

	// Create the result with the mock output
	result := &ExecutionResult{
		Stdout:      fmt.Sprintf("[Vertex AI] Executed:\n%s\nOutput: Code executed successfully in Vertex AI", input.Code),
		Stderr:      "",
		OutputFiles: []File{},
	}

	// Publish event after execution
	if invocationContext.Events != nil {
		invocationContext.Events.Publish(ctx, events.ToolResultReceived, map[string]interface{}{
			"tool":   "vertex_ai_code_executor",
			"result": result,
		})
	}

	return result, nil
}

// GetConfig returns the configuration for this executor
func (e *VertexAICodeExecutor) GetConfig() CodeExecConfig {
	return e.config
}

// Cleanup cleans up any resources used by the executor
func (e *VertexAICodeExecutor) Cleanup() error {
	// No resources to clean up for Vertex AI executor
	return nil
}
