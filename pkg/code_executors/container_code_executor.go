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
	"time"

	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/telemetry"
)

const DefaultImageTag = "adk-code-executor:latest"

// ContainerCodeExecutor is a code executor that runs code in a Docker container.
type ContainerCodeExecutor struct {
	config        CodeExecConfig
	image         string
	dockerPath    string
	containerName string
	dockerClient  DockerClient
	initialized   bool
}

// ContainerCodeExecutorOption is a functional option for ContainerCodeExecutor
type ContainerCodeExecutorOption func(*ContainerCodeExecutor)

// WithImage sets the image for the container
func WithImage(image string) ContainerCodeExecutorOption {
	return func(e *ContainerCodeExecutor) {
		e.image = image
	}
}

// WithDockerPath sets the path to the Dockerfile
func WithDockerPath(dockerPath string) ContainerCodeExecutorOption {
	return func(e *ContainerCodeExecutor) {
		e.dockerPath = dockerPath
	}
}

// WithContainerCodeExecConfig sets the config for the container code executor
func WithContainerCodeExecConfig(config CodeExecConfig) ContainerCodeExecutorOption {
	return func(e *ContainerCodeExecutor) {
		e.config = config
	}
}

// NewContainerCodeExecutor creates a new ContainerCodeExecutor
func NewContainerCodeExecutor(opts ...ContainerCodeExecutorOption) (*ContainerCodeExecutor, error) {
	executor := &ContainerCodeExecutor{
		config:        DefaultCodeExecConfig(),
		image:         DefaultImageTag,
		containerName: fmt.Sprintf("adk-code-executor-%d", time.Now().Unix()),
		dockerClient:  &mockDockerClient{}, // For now, use a mock client that doesn't actually interact with Docker
	}

	// Apply options
	for _, opt := range opts {
		opt(executor)
	}

	// Validation
	if executor.image == "" && executor.dockerPath == "" {
		return nil, errors.New("either image or dockerPath must be set for ContainerCodeExecutor")
	}
	if executor.config.Stateful {
		return nil, errors.New("cannot set `Stateful=true` in ContainerCodeExecutor")
	}
	if executor.config.OptimizeDataFile {
		return nil, errors.New("cannot set `OptimizeDataFile=true` in ContainerCodeExecutor")
	}

	// Force these settings for safety
	executor.config.Stateful = false
	executor.config.OptimizeDataFile = false

	return executor, nil
}

// ExecuteCode executes code in a container and returns the result
func (e *ContainerCodeExecutor) ExecuteCode(
	invocationContext *InvocationContext,
	input *CodeExecutionInput,
) (*ExecutionResult, error) {
	// Create a span for tracking this execution
	ctx, span := telemetry.StartSpan(invocationContext.Context, "ContainerCodeExecutor.ExecuteCode")
	defer span.End()

	// Publish event before execution
	if invocationContext.Events != nil {
		invocationContext.Events.Publish(ctx, events.ToolCalled, map[string]interface{}{
			"tool": "container_executor",
			"code": input.Code,
		})
	}

	// Initialize the container if needed
	if !e.initialized {
		if err := e.initialize(); err != nil {
			span.SetAttribute("error", err.Error())
			if invocationContext.Events != nil {
				invocationContext.Events.Publish(ctx, events.ToolError, map[string]interface{}{
					"tool":  "container_executor",
					"error": err.Error(),
				})
			}
			return nil, fmt.Errorf("failed to initialize container: %w", err)
		}
	}

	// Mock execution for now
	// In a real implementation, this would use Docker SDK to execute the code

	// This is a mock implementation that echoes the code back
	stdout := fmt.Sprintf("Executed in container %s:\n%s", e.containerName, input.Code)
	stderr := ""

	// If the code contains "error", simulate an error
	if strings.Contains(strings.ToLower(input.Code), "error") {
		stderr = "Error executing code in container"
		stdout = ""
	}

	// Create the result
	result := &ExecutionResult{
		Stdout:      stdout,
		Stderr:      stderr,
		OutputFiles: []File{},
	}

	// Publish event after execution
	if invocationContext.Events != nil {
		invocationContext.Events.Publish(ctx, events.ToolResultReceived, map[string]interface{}{
			"tool":   "container_executor",
			"result": result,
		})
	}

	return result, nil
}

// GetConfig returns the configuration for this executor
func (e *ContainerCodeExecutor) GetConfig() CodeExecConfig {
	return e.config
}

// initialize sets up the container for execution
func (e *ContainerCodeExecutor) initialize() error {
	// In a real implementation, this would:
	// 1. Build the Docker image if dockerPath is set
	// 2. Create and start the container
	// 3. Verify Python is installed

	// For now, just set the initialized flag
	e.initialized = true
	return nil
}

// Cleanup stops and removes the container
func (e *ContainerCodeExecutor) Cleanup() error {
	// In a real implementation, this would stop and remove the container

	// Reset initialized state
	e.initialized = false
	return nil
}

// DockerClient interface for interacting with Docker
type DockerClient interface {
	// Add methods as needed for Docker interactions
}

// mockDockerClient implements DockerClient for testing
type mockDockerClient struct{}

// Implement DockerClient methods as needed
