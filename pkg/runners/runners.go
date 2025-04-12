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

// Package runners provides functionality for running agents.
package runners

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/adk-golang/pkg/agents"
	"github.com/google/adk-golang/pkg/telemetry"
)

// Runner is an interface for running agents.
type Runner interface {
	// Run runs the agent with the given input and produces output.
	Run(ctx context.Context, agent *agents.Agent, input string) (string, error)

	// RunInteractive runs the agent in an interactive mode, reading from in and writing to out.
	RunInteractive(ctx context.Context, agent *agents.Agent, in io.Reader, out io.Writer) error
}

// SimpleRunner is a basic implementation of Runner.
type SimpleRunner struct{}

// NewSimpleRunner creates a new SimpleRunner.
func NewSimpleRunner() *SimpleRunner {
	return &SimpleRunner{}
}

// Run runs the agent with the given input and produces output.
func (r *SimpleRunner) Run(ctx context.Context, agent *agents.Agent, input string) (string, error) {
	if agent == nil {
		return "", errors.New("agent cannot be nil")
	}

	// Create a span for this runner execution
	ctx, span := telemetry.StartSpan(ctx, "SimpleRunner.Run")
	defer span.End()

	// Add agent metadata as span attributes
	span.SetAttribute("agent.name", agent.Name())
	span.SetAttribute("agent.model", agent.Model())

	// Process the input with the agent
	response, err := agent.Process(ctx, input)
	if err != nil {
		span.SetAttribute("error", err.Error())
		return "", err
	}

	return response, nil
}

// RunInteractive runs the agent in an interactive mode, reading from in and writing to out.
func (r *SimpleRunner) RunInteractive(ctx context.Context, agent *agents.Agent, in io.Reader, out io.Writer) error {
	if agent == nil {
		return errors.New("agent cannot be nil")
	}

	// Create a span for this interactive session
	ctx, span := telemetry.StartSpan(ctx, "SimpleRunner.RunInteractive")
	defer span.End()

	// Add agent metadata as span attributes
	span.SetAttribute("agent.name", agent.Name())
	span.SetAttribute("agent.model", agent.Model())

	// Print welcome message
	fmt.Fprintf(out, "Starting interactive session with %s agent\n", agent.Name())
	fmt.Fprintf(out, "Type 'exit' or 'quit' to end the session\n\n")

	// Input buffer
	buf := make([]byte, 1024)

	for {
		// Prompt for input
		fmt.Fprintf(out, "> ")

		// Read user input
		n, err := in.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			span.SetAttribute("error", err.Error())
			return err
		}

		// Convert to string and trim spaces
		input := string(buf[:n])
		input = input[:len(input)-1] // Remove newline

		// Check for exit command
		if input == "exit" || input == "quit" {
			return nil
		}

		// Create a span for this interaction
		interactionCtx, interactionSpan := telemetry.StartSpan(ctx, "agent.interaction")
		interactionSpan.SetAttribute("input", input)

		// Process the input
		response, err := agent.Process(interactionCtx, input)
		if err != nil {
			interactionSpan.SetAttribute("error", err.Error())
			fmt.Fprintf(out, "Error: %v\n", err)
			interactionSpan.End()
			continue
		}

		// Track the response
		interactionSpan.SetAttribute("response_length", fmt.Sprintf("%d", len(response)))
		interactionSpan.End()

		// Write the response
		fmt.Fprintf(out, "%s\n", response)
	}
}
