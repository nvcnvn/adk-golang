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

// ExitLoopTool is a tool that allows an agent to exit a loop.
var ExitLoopTool = NewTool(
	"exit_loop",
	"Exit the current loop. Call this function only when you are instructed to do so.",
	ToolSchema{
		Input: ParameterSchema{
			Type:       "object",
			Properties: map[string]ParameterSchema{},
		},
		Output: map[string]ParameterSchema{
			"success": {
				Type:        "boolean",
				Description: "Whether the exit was successful",
			},
		},
	},
	func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		// In a real implementation, this would set a flag in the execution context
		// to indicate that the loop should exit
		// For now we just return success
		return map[string]interface{}{
			"success": true,
			"message": "Loop exit signal sent",
		}, nil
	},
)
