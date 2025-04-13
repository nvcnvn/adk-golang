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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/nvcnvn/adk-golang/pkg/agents"
	"github.com/nvcnvn/adk-golang/pkg/telemetry"
)

// Interaction represents a single interaction between a user and an agent.
type Interaction struct {
	Input     string    `json:"input"`
	Response  string    `json:"response"`
	Timestamp time.Time `json:"timestamp"`
}

// Session represents an interactive session with an agent.
type Session struct {
	AgentName    string        `json:"agent_name"`
	AgentModel   string        `json:"agent_model"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time,omitempty"`
	Interactions []Interaction `json:"interactions"`
}

// Runner is an interface for running agents.
type Runner interface {
	// Run runs the agent with the given input and produces output.
	Run(ctx context.Context, agent *agents.Agent, input string) (string, error)

	// RunInteractive runs the agent in an interactive mode, reading from in and writing to out.
	RunInteractive(ctx context.Context, agent *agents.Agent, in io.Reader, out io.Writer) error

	// SetSaveSessionEnabled enables or disables session saving.
	SetSaveSessionEnabled(enabled bool)
}

// SimpleRunner is a basic implementation of Runner.
type SimpleRunner struct {
	saveSession bool
	session     Session
}

// NewSimpleRunner creates a new SimpleRunner.
func NewSimpleRunner() *SimpleRunner {
	return &SimpleRunner{
		saveSession: false,
		session: Session{
			StartTime:    time.Now(),
			Interactions: []Interaction{},
		},
	}
}

// SetSaveSessionEnabled enables or disables session saving.
func (r *SimpleRunner) SetSaveSessionEnabled(enabled bool) {
	r.saveSession = enabled
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

	// Track the interaction if session saving is enabled
	if r.saveSession {
		r.trackInteraction(input, response)
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

	// Initialize session data if saving is enabled
	if r.saveSession {
		r.session.AgentName = agent.Name()
		r.session.AgentModel = agent.Model()
		r.session.StartTime = time.Now()
	}

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
				break
			}
			span.SetAttribute("error", err.Error())
			return err
		}

		// Convert to string and trim spaces
		input := string(buf[:n])
		input = input[:len(input)-1] // Remove newline

		// Check for exit command
		if input == "exit" || input == "quit" {
			break
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

		// Track the interaction if session saving is enabled
		if r.saveSession {
			r.trackInteraction(input, response)
		}
	}

	// Save the session if enabled
	if r.saveSession {
		r.session.EndTime = time.Now()
		if err := r.saveSessionToFile(agent.Name()); err != nil {
			fmt.Fprintf(out, "Failed to save session: %v\n", err)
		} else {
			fmt.Fprintf(out, "Session saved to file\n")
		}
	}

	return nil
}

// trackInteraction adds an interaction to the current session.
func (r *SimpleRunner) trackInteraction(input, response string) {
	r.session.Interactions = append(r.session.Interactions, Interaction{
		Input:     input,
		Response:  response,
		Timestamp: time.Now(),
	})
}

// saveSessionToFile saves the current session to a JSON file.
func (r *SimpleRunner) saveSessionToFile(agentName string) error {
	// Create sessions directory if it doesn't exist
	sessionsDir := filepath.Join(".", "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return err
	}

	// Create a filename based on agent name and timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(sessionsDir, fmt.Sprintf("%s_session_%s.json", agentName, timestamp))

	// Marshal session data to JSON
	data, err := json.MarshalIndent(r.session, "", "  ")
	if err != nil {
		return err
	}

	// Write the JSON data to file
	return os.WriteFile(filename, data, 0644)
}
