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
)

// TransferToAgentTool is a tool that allows an agent to transfer control to another agent.
var TransferToAgentTool = NewTool(
	"transfer_to_agent",
	"Transfer the conversation to another agent",
	ToolSchema{
		Input: ParameterSchema{
			Type: "object",
			Properties: map[string]ParameterSchema{
				"agent_name": {
					Type:        "string",
					Description: "The name of the agent to transfer to",
					Required:    true,
				},
			},
		},
		Output: map[string]ParameterSchema{
			"success": {
				Type:        "boolean",
				Description: "Whether the transfer was successful",
			},
		},
	},
	func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		agentName, ok := input["agent_name"].(string)
		if !ok {
			return map[string]interface{}{
				"success": false,
				"error":   "agent_name must be a string",
			}, nil
		}

		// Set transfer info to context
		// In a real implementation, this would use proper context values
		// that are read by the agent system later
		// For now we just return the agent name so it can be handled elsewhere
		return map[string]interface{}{
			"success":      true,
			"target_agent": agentName,
		}, nil
	},
)
