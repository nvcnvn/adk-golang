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
	"context"

	"github.com/nvcnvn/adk-golang/pkg/events"
)

// EventPublisher defines an interface for publishing events
type EventPublisher interface {
	Publish(ctx context.Context, eventType events.EventType, payload interface{})
}

// CodeBlockDelimiter represents the start and end delimiters for code blocks
type CodeBlockDelimiter struct {
	Start string
	End   string
}

// ExecutionResultDelimiter represents the start and end delimiters for execution results
type ExecutionResultDelimiter struct {
	Start string
	End   string
}

// CodeExecutionInput represents the input for code execution
type CodeExecutionInput struct {
	Code       string
	InputFiles []File
}

// CodeExecConfig represents the configuration for a code executor
type CodeExecConfig struct {
	OptimizeDataFile         bool
	Stateful                 bool
	ErrorRetryAttempts       int
	CodeBlockDelimiters      []CodeBlockDelimiter
	ExecutionResultDelimiter ExecutionResultDelimiter
}

// DefaultCodeExecConfig returns the default configuration for a code executor
func DefaultCodeExecConfig() CodeExecConfig {
	return CodeExecConfig{
		OptimizeDataFile:   false,
		Stateful:           false,
		ErrorRetryAttempts: 2,
		CodeBlockDelimiters: []CodeBlockDelimiter{
			{Start: "```tool_code\n", End: "\n```"},
			{Start: "```python\n", End: "\n```"},
		},
		ExecutionResultDelimiter: ExecutionResultDelimiter{
			Start: "```tool_output\n",
			End:   "\n```",
		},
	}
}

// InvocationContext represents the context for a code execution invocation
type InvocationContext struct {
	InvocationID string
	Context      context.Context
	Events       EventPublisher
}

// BaseCodeExecutor defines the interface for executing code
type BaseCodeExecutor interface {
	// ExecuteCode executes code and returns the code execution result
	ExecuteCode(invocationContext *InvocationContext, input *CodeExecutionInput) (*ExecutionResult, error)

	// GetConfig returns the configuration for the code executor
	GetConfig() CodeExecConfig

	// Cleanup cleans up any resources used by the code executor
	Cleanup() error
}
