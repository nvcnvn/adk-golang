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
)

// UnsafeLocalCodeExecutor is a code executor that runs code locally with no security sandbox.
// This should only be used in trusted environments with trusted code.
type UnsafeLocalCodeExecutor struct {
	config CodeExecConfig
}

// NewUnsafeLocalCodeExecutor creates a new UnsafeLocalCodeExecutor
func NewUnsafeLocalCodeExecutor(opts ...UnsafeLocalCodeExecutorOption) (*UnsafeLocalCodeExecutor, error) {
	// Set default configuration
	executor := &UnsafeLocalCodeExecutor{
		config: DefaultCodeExecConfig(),
	}

	// Override with any options
	for _, opt := range opts {
		opt(executor)
	}

	// Validation
	if executor.config.Stateful {
		return nil, errors.New("cannot set `Stateful=true` in UnsafeLocalCodeExecutor")
	}
	if executor.config.OptimizeDataFile {
		return nil, errors.New("cannot set `OptimizeDataFile=true` in UnsafeLocalCodeExecutor")
	}

	// Force these settings for safety
	executor.config.Stateful = false
	executor.config.OptimizeDataFile = false

	return executor, nil
}

// UnsafeLocalCodeExecutorOption is a functional option for UnsafeLocalCodeExecutor
type UnsafeLocalCodeExecutorOption func(*UnsafeLocalCodeExecutor)

// WithCodeBlockDelimiters sets the code block delimiters
func WithCodeBlockDelimiters(delimiters []CodeBlockDelimiter) UnsafeLocalCodeExecutorOption {
	return func(e *UnsafeLocalCodeExecutor) {
		e.config.CodeBlockDelimiters = delimiters
	}
}

// WithExecutionResultDelimiter sets the execution result delimiter
func WithExecutionResultDelimiter(delimiter ExecutionResultDelimiter) UnsafeLocalCodeExecutorOption {
	return func(e *UnsafeLocalCodeExecutor) {
		e.config.ExecutionResultDelimiter = delimiter
	}
}

// WithErrorRetryAttempts sets the number of error retry attempts
func WithErrorRetryAttempts(attempts int) UnsafeLocalCodeExecutorOption {
	return func(e *UnsafeLocalCodeExecutor) {
		e.config.ErrorRetryAttempts = attempts
	}
}

// ExecuteCode executes code locally and returns the result
func (e *UnsafeLocalCodeExecutor) ExecuteCode(
	invocationContext *InvocationContext,
	input *CodeExecutionInput,
) (*ExecutionResult, error) {
	// This is the unsafe local executor, which would evaluate code in the current process
	// In Go, we don't have a direct equivalent to Python's exec() for safety reasons
	// For practical purposes, we'll redirect to the language-specific executors

	// Determine language from the first code block delimiter that matches
	language := "unknown"
	for _, delimiter := range e.config.CodeBlockDelimiters {
		if len(delimiter.Start) >= 4 && delimiter.Start[:4] == "```" {
			lang := delimiter.Start[3:]
			if lang == "python\n" {
				language = "python"
				break
			} else if lang == "javascript\n" || lang == "js\n" {
				language = "javascript"
				break
			} else if lang == "tool_code\n" {
				// Default to python for tool_code
				language = "python"
				break
			}
		}
	}

	// Create appropriate executor based on language
	var executor CodeExecutor
	var err error

	switch language {
	case "python":
		executor, err = NewPythonExecutor()
	case "javascript":
		executor, err = NewJavaScriptExecutor()
	default:
		return nil, fmt.Errorf("unsupported language detected for UnsafeLocalCodeExecutor: %s", language)
	}

	if err != nil {
		return nil, err
	}

	// Delegate execution to the language-specific executor
	result, err := executor.Execute(invocationContext.Context, input.Code, input.InputFiles)
	if err != nil {
		return nil, err
	}

	// Clean up resources when done
	if baseExecutor, ok := executor.(interface{ Cleanup() error }); ok {
		defer baseExecutor.Cleanup()
	}

	return result, nil
}

// GetConfig returns the configuration for this executor
func (e *UnsafeLocalCodeExecutor) GetConfig() CodeExecConfig {
	return e.config
}

// Cleanup cleans up any resources used by the executor
func (e *UnsafeLocalCodeExecutor) Cleanup() error {
	// No resources to clean up
	return nil
}
