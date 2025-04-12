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
)

// SequentialAgent runs its sub-agents in sequence, passing the output of one to the next.
type SequentialAgent struct {
	Agent
	subAgents []*Agent
}

// SequentialAgentConfig holds configuration for creating a SequentialAgent.
type SequentialAgentConfig struct {
	Name        string
	Description string
	SubAgents   []*Agent
}

// NewSequentialAgent creates a new agent that processes sub-agents in sequence.
func NewSequentialAgent(config SequentialAgentConfig) *SequentialAgent {
	return &SequentialAgent{
		Agent: Agent{
			name:        config.Name,
			description: config.Description,
		},
		subAgents: config.SubAgents,
	}
}

// Process handles a message by passing it through each sub-agent in sequence.
func (a *SequentialAgent) Process(ctx context.Context, message string) (string, error) {
	currentMessage := message
	var response string
	var err error

	// Process through each sub-agent in sequence
	for _, subAgent := range a.subAgents {
		currentMessage, err = subAgent.Process(ctx, currentMessage)
		if err != nil {
			return "", err
		}
	}

	response = currentMessage
	return response, nil
}

// SubAgents returns the sub-agents of this sequential agent.
func (a *SequentialAgent) SubAgents() []*Agent {
	return a.subAgents
}
