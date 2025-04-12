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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/telemetry"
)

// File represents a file with its name and content.
type File struct {
	Name    string
	Content []byte
}

// ExecutionResult contains the output of a code execution.
type ExecutionResult struct {
	Stdout      string
	Stderr      string
	OutputFiles []File
}

// CodeExecutor defines the interface for executing code.
type CodeExecutor interface {
	// Execute executes the given code and returns the result.
	Execute(ctx context.Context, code string, files []File) (*ExecutionResult, error)
}

// BaseExecutor provides common functionality for code execution.
type BaseExecutor struct {
	TempDir string
}

// NewBaseExecutor creates a new BaseExecutor.
func NewBaseExecutor() (*BaseExecutor, error) {
	// Create a temporary directory for code execution
	tempDir, err := os.MkdirTemp("", "adk-code-executor-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	return &BaseExecutor{
		TempDir: tempDir,
	}, nil
}

// Cleanup removes the temporary directory.
func (e *BaseExecutor) Cleanup() error {
	if e.TempDir != "" {
		return os.RemoveAll(e.TempDir)
	}
	return nil
}

// SaveFile saves the given file to the temporary directory.
func (e *BaseExecutor) SaveFile(file File) (string, error) {
	path := filepath.Join(e.TempDir, file.Name)

	// Create parent directories if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, file.Content, 0644); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return path, nil
}

// PythonExecutor executes Python code.
type PythonExecutor struct {
	BaseExecutor
}

// NewPythonExecutor creates a new PythonExecutor.
func NewPythonExecutor() (*PythonExecutor, error) {
	base, err := NewBaseExecutor()
	if err != nil {
		return nil, err
	}

	return &PythonExecutor{
		BaseExecutor: *base,
	}, nil
}

// Execute executes Python code.
func (e *PythonExecutor) Execute(ctx context.Context, code string, files []File) (*ExecutionResult, error) {
	// Create a span for tracking this execution
	ctx, span := telemetry.StartSpan(ctx, "PythonExecutor.Execute")
	defer span.End()

	// Save the code to a temporary file
	codePath := filepath.Join(e.TempDir, "script.py")
	if err := os.WriteFile(codePath, []byte(code), 0644); err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to write Python script: %w", err)
	}

	// Save any input files
	for i, file := range files {
		path, err := e.SaveFile(file)
		if err != nil {
			span.SetAttribute("error", err.Error())
			return nil, err
		}
		span.SetAttribute(fmt.Sprintf("input_file_%d", i), path)
	}

	// Build the command
	cmd := exec.CommandContext(ctx, "python3", codePath)
	cmd.Dir = e.TempDir

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Publish event before execution
	events.Publish(events.ToolCalled, map[string]interface{}{
		"tool": "python_executor",
		"code": code,
	})

	// Run the command
	if err := cmd.Start(); err != nil {
		span.SetAttribute("error", err.Error())
		events.Publish(events.ToolError, map[string]interface{}{
			"tool":  "python_executor",
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to start Python process: %w", err)
	}

	// Read stdout and stderr
	stdoutBytes, err := io.ReadAll(stdout)
	if err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to read stdout: %w", err)
	}

	stderrBytes, err := io.ReadAll(stderr)
	if err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to read stderr: %w", err)
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		span.SetAttribute("error", err.Error())
		events.Publish(events.ToolError, map[string]interface{}{
			"tool":   "python_executor",
			"error":  err.Error(),
			"stdout": string(stdoutBytes),
			"stderr": string(stderrBytes),
		})
	}

	// Check for output files (files that were created or modified during execution)
	outputFiles, err := e.collectOutputFiles()
	if err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to collect output files: %w", err)
	}

	// Create the result
	result := &ExecutionResult{
		Stdout:      string(stdoutBytes),
		Stderr:      string(stderrBytes),
		OutputFiles: outputFiles,
	}

	// Publish event after execution
	events.Publish(events.ToolResultReceived, map[string]interface{}{
		"tool":   "python_executor",
		"result": result,
	})

	return result, nil
}

// collectOutputFiles finds files that were created or modified during execution.
func (e *PythonExecutor) collectOutputFiles() ([]File, error) {
	var outputFiles []File

	err := filepath.Walk(e.TempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and the script file itself
		if info.IsDir() || filepath.Base(path) == "script.py" {
			return nil
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Get the relative path
		relPath, err := filepath.Rel(e.TempDir, path)
		if err != nil {
			return err
		}

		// Add to output files
		outputFiles = append(outputFiles, File{
			Name:    relPath,
			Content: content,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return outputFiles, nil
}

// JavaScriptExecutor executes JavaScript code using Node.js.
type JavaScriptExecutor struct {
	BaseExecutor
}

// NewJavaScriptExecutor creates a new JavaScriptExecutor.
func NewJavaScriptExecutor() (*JavaScriptExecutor, error) {
	base, err := NewBaseExecutor()
	if err != nil {
		return nil, err
	}

	return &JavaScriptExecutor{
		BaseExecutor: *base,
	}, nil
}

// Execute executes JavaScript code.
func (e *JavaScriptExecutor) Execute(ctx context.Context, code string, files []File) (*ExecutionResult, error) {
	// Create a span for tracking this execution
	ctx, span := telemetry.StartSpan(ctx, "JavaScriptExecutor.Execute")
	defer span.End()

	// Save the code to a temporary file
	codePath := filepath.Join(e.TempDir, "script.js")
	if err := os.WriteFile(codePath, []byte(code), 0644); err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to write JavaScript script: %w", err)
	}

	// Save any input files
	for i, file := range files {
		path, err := e.SaveFile(file)
		if err != nil {
			span.SetAttribute("error", err.Error())
			return nil, err
		}
		span.SetAttribute(fmt.Sprintf("input_file_%d", i), path)
	}

	// Build the command
	cmd := exec.CommandContext(ctx, "node", codePath)
	cmd.Dir = e.TempDir

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Publish event before execution
	events.Publish(events.ToolCalled, map[string]interface{}{
		"tool": "javascript_executor",
		"code": code,
	})

	// Run the command
	if err := cmd.Start(); err != nil {
		span.SetAttribute("error", err.Error())
		events.Publish(events.ToolError, map[string]interface{}{
			"tool":  "javascript_executor",
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to start Node.js process: %w", err)
	}

	// Read stdout and stderr
	stdoutBytes, err := io.ReadAll(stdout)
	if err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to read stdout: %w", err)
	}

	stderrBytes, err := io.ReadAll(stderr)
	if err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to read stderr: %w", err)
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		span.SetAttribute("error", err.Error())
		events.Publish(events.ToolError, map[string]interface{}{
			"tool":   "javascript_executor",
			"error":  err.Error(),
			"stdout": string(stdoutBytes),
			"stderr": string(stderrBytes),
		})
	}

	// Check for output files
	outputFiles, err := e.collectOutputFiles()
	if err != nil {
		span.SetAttribute("error", err.Error())
		return nil, fmt.Errorf("failed to collect output files: %w", err)
	}

	// Create the result
	result := &ExecutionResult{
		Stdout:      string(stdoutBytes),
		Stderr:      string(stderrBytes),
		OutputFiles: outputFiles,
	}

	// Publish event after execution
	events.Publish(events.ToolResultReceived, map[string]interface{}{
		"tool":   "javascript_executor",
		"result": result,
	})

	return result, nil
}

// collectOutputFiles finds files that were created or modified during execution.
func (e *JavaScriptExecutor) collectOutputFiles() ([]File, error) {
	var outputFiles []File

	err := filepath.Walk(e.TempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and the script file itself
		if info.IsDir() || filepath.Base(path) == "script.js" {
			return nil
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Get the relative path
		relPath, err := filepath.Rel(e.TempDir, path)
		if err != nil {
			return err
		}

		// Add to output files
		outputFiles = append(outputFiles, File{
			Name:    relPath,
			Content: content,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return outputFiles, nil
}

// NewCodeExecutor creates a new code executor for the given language.
func NewCodeExecutor(language string) (CodeExecutor, error) {
	language = strings.ToLower(language)

	switch language {
	case "python", "py":
		return NewPythonExecutor()
	case "javascript", "js":
		return NewJavaScriptExecutor()
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}
