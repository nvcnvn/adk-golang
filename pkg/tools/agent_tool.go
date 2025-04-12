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

package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// WrappableAgent is an interface that represents an agent that can be wrapped as a tool.
// This avoids direct imports of the agents package which would create an import cycle.
type WrappableAgent interface {
	Name() string
	Description() string
	Process(ctx context.Context, message string) (string, error)
}

// AgentTool is a tool that wraps an agent.
// This tool allows an agent to be called as a tool within a larger application.
type AgentTool struct {
	*LlmToolAdaptor
	agent             WrappableAgent
	skipSummarization bool
}

// AgentToolConfig contains configuration options for an AgentTool.
type AgentToolConfig struct {
	// SkipSummarization indicates whether to skip summarization of the agent output.
	SkipSummarization bool
}

// NewAgentTool creates a new tool that wraps an agent.
func NewAgentTool(agent WrappableAgent, config *AgentToolConfig) *AgentTool {
	// Create a base tool with execute function that delegates to our agent
	baseTool := &BaseTool{
		name:        agent.Name(),
		description: agent.Description(),
		schema: ToolSchema{
			// The input schema is dynamically determined based on the agent's configuration
			Input: ParameterSchema{
				Type: "object",
				Properties: map[string]ParameterSchema{
					"request": {
						Type:        "string",
						Description: "The request to send to the agent",
						Required:    true,
					},
				},
			},
			Output: map[string]ParameterSchema{
				"response": {
					Type:        "string",
					Description: "The response from the agent",
				},
			},
		},
		executeFn: nil, // Will be set below
	}

	skipSum := false
	if config != nil {
		skipSum = config.SkipSummarization
	}

	agentTool := &AgentTool{
		LlmToolAdaptor:    NewLlmToolAdaptor(baseTool, false),
		agent:             agent,
		skipSummarization: skipSum,
	}

	// Set the execute function now that we have the agentTool instance
	baseTool.executeFn = agentTool.execute

	return agentTool
}

// execute runs the wrapped agent with the given input
func (a *AgentTool) execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// Get the request from the input
	var request string
	if req, ok := input["request"].(string); ok {
		request = req
	} else {
		// Try to convert the entire input map to JSON if it's not a simple string request
		inputJSON, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input: %v", err)
		}
		request = string(inputJSON)
	}

	// Directly use the agent's Process method
	response, err := a.agent.Process(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to process agent request: %v", err)
	}

	// Return the result
	return map[string]interface{}{
		"response": response,
	}, nil
}

// SetSkipSummarization sets whether to skip summarization of the agent output
func (a *AgentTool) SetSkipSummarization(skip bool) {
	a.skipSummarization = skip
}
