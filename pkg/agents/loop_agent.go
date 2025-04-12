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
	"fmt"
)

// LoopAgent runs its sub-agents repeatedly until a condition is met or max iterations is reached.
type LoopAgent struct {
	Agent
	subAgents     []*Agent
	maxIterations int
}

// LoopAgentConfig holds configuration for creating a LoopAgent.
type LoopAgentConfig struct {
	Name          string
	Description   string
	SubAgents     []*Agent
	MaxIterations int
}

// NewLoopAgent creates a new agent that processes sub-agents in a loop.
func NewLoopAgent(config LoopAgentConfig) *LoopAgent {
	// If max iterations not specified, default to 10 to prevent infinite loops
	maxIter := config.MaxIterations
	if maxIter <= 0 {
		maxIter = 10
	}

	return &LoopAgent{
		Agent: Agent{
			name:        config.Name,
			description: config.Description,
		},
		subAgents:     config.SubAgents,
		maxIterations: maxIter,
	}
}

// Process handles a message by processing it through all sub-agents repeatedly.
func (a *LoopAgent) Process(ctx context.Context, message string) (string, error) {
	currentMessage := message
	var err error
	iterations := 0

	// Continue looping until max iterations is reached
	for iterations < a.maxIterations {
		iterations++

		// Process through each sub-agent in sequence
		for _, subAgent := range a.subAgents {
			currentMessage, err = subAgent.Process(ctx, currentMessage)
			if err != nil {
				return "", err
			}

			// Check for context cancellation
			select {
			case <-ctx.Done():
				return currentMessage, fmt.Errorf("loop agent terminated: %v", ctx.Err())
			default:
				// Continue processing
			}
		}
	}

	return currentMessage, nil
}

// SubAgents returns the sub-agents of this loop agent.
func (a *LoopAgent) SubAgents() []*Agent {
	return a.subAgents
}

// MaxIterations returns the maximum number of iterations for this loop agent.
func (a *LoopAgent) MaxIterations() int {
	return a.maxIterations
}
