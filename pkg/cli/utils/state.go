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

// Package utils provides utility functions for the CLI
package utils

import (
	"regexp"
)

// createEmptyState populates a map with empty string values for parameters
// found in the agent's instruction using regex
func createEmptyState(agent interface{}, allState map[string]interface{}) {
	// Process Agent type
	if a, ok := agent.(*struct {
		Instruction func() string
		SubAgents   func() []*struct{}
	}); ok && a != nil {
		instruction := a.Instruction()
		if instruction != "" {
			findStateParams(instruction, allState)
		}

		// Process sub-agents if available
		if subAgents := a.SubAgents(); subAgents != nil {
			for _, subAgent := range subAgents {
				if subAgent != nil {
					createEmptyState(subAgent, allState)
				}
			}
		}
	}

	// Process LlmAgent type
	type llmAgentType struct {
		SystemInstructions string
	}
	if a, ok := agent.(*llmAgentType); ok && a != nil && a.SystemInstructions != "" {
		findStateParams(a.SystemInstructions, allState)
	}
}

// findStateParams finds parameters in format {paramName} in the text
// and adds them with empty string values to the provided map
func findStateParams(text string, allState map[string]interface{}) {
	re := regexp.MustCompile(`{([\w]+)}`)
	matches := re.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			key := match[1]
			allState[key] = ""
		}
	}
}

// CreateEmptyState creates empty string values for non-initialized states in agent instructions
// Similar to the Python implementation, it extracts parameter names from instruction strings
func CreateEmptyState(agent interface{}, initializedStates map[string]interface{}) map[string]interface{} {
	// Create a map for non-initialized states
	nonInitializedStates := make(map[string]interface{})

	// Populate non-initialized states from agent and its sub-agents
	createEmptyState(agent, nonInitializedStates)

	// Remove states that are already initialized
	if initializedStates != nil {
		for key := range initializedStates {
			delete(nonInitializedStates, key)
		}
	}

	return nonInitializedStates
}
