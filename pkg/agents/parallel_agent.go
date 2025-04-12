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

// Package agents provides the core agent types and functionality.
package agents

import (
	"context"
	"strings"
	"sync"
)

// ParallelAgent runs its sub-agents in parallel and aggregates their responses.
type ParallelAgent struct {
	Agent
	subAgents []*Agent
}

// ParallelAgentConfig holds configuration for creating a ParallelAgent.
type ParallelAgentConfig struct {
	Name        string
	Description string
	SubAgents   []*Agent
}

// NewParallelAgent creates a new agent that processes sub-agents in parallel.
func NewParallelAgent(config ParallelAgentConfig) *ParallelAgent {
	return &ParallelAgent{
		Agent: Agent{
			name:        config.Name,
			description: config.Description,
		},
		subAgents: config.SubAgents,
	}
}

// Process handles a message by processing it through all sub-agents in parallel.
func (a *ParallelAgent) Process(ctx context.Context, message string) (string, error) {
	var wg sync.WaitGroup
	responses := make([]string, len(a.subAgents))
	errors := make([]error, len(a.subAgents))

	// Process through each sub-agent in parallel
	for i, subAgent := range a.subAgents {
		wg.Add(1)
		go func(idx int, agent *Agent) {
			defer wg.Done()
			resp, err := agent.Process(ctx, message)
			responses[idx] = resp
			errors[idx] = err
		}(i, subAgent)
	}

	// Wait for all sub-agents to complete
	wg.Wait()

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return "", err
		}
	}

	// Combine responses
	return strings.Join(responses, "\n\n"), nil
}

// SubAgents returns the sub-agents of this parallel agent.
func (a *ParallelAgent) SubAgents() []*Agent {
	return a.subAgents
}
