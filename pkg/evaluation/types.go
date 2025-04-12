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

package evaluation

import (
	"encoding/json"
	"fmt"
)

// EvaluationEntry represents a single entry in the evaluation dataset
type EvaluationEntry map[string]interface{}

// EvaluationConversation represents a conversation containing multiple evaluation entries
type EvaluationConversation []EvaluationEntry

// EvaluationDataset represents a collection of evaluation conversations
type EvaluationDataset []EvaluationConversation

// ToolUse represents a tool usage instance in the evaluation
type ToolUse struct {
	ToolName   string                 `json:"tool_name"`
	ToolInput  map[string]interface{} `json:"tool_input"`
	ToolOutput interface{}            `json:"mock_tool_output,omitempty"`
}

// GetQuery returns the query from an evaluation entry
func (e EvaluationEntry) GetQuery() string {
	if query, ok := e[Query].(string); ok {
		return query
	}
	return ""
}

// GetResponse returns the response from an evaluation entry
func (e EvaluationEntry) GetResponse() string {
	if response, ok := e[Response].(string); ok {
		return response
	}
	return ""
}

// GetReference returns the reference from an evaluation entry
func (e EvaluationEntry) GetReference() string {
	if reference, ok := e[Reference].(string); ok {
		return reference
	}
	return ""
}

// GetExpectedToolUse returns the expected tool uses from an evaluation entry
func (e EvaluationEntry) GetExpectedToolUse() ([]ToolUse, error) {
	expectedToolUseI, exists := e[ExpectedToolUse]
	if !exists {
		return nil, nil
	}

	var toolUses []ToolUse

	// Convert from interface{} to []ToolUse
	expectedToolUseBytes, err := json.Marshal(expectedToolUseI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal expected tool use: %v", err)
	}

	if err := json.Unmarshal(expectedToolUseBytes, &toolUses); err != nil {
		return nil, fmt.Errorf("failed to unmarshal expected tool use: %v", err)
	}

	return toolUses, nil
}

// GetActualToolUse returns the actual tool uses from an evaluation entry
func (e EvaluationEntry) GetActualToolUse() ([]ToolUse, error) {
	actualToolUseI, exists := e[ActualToolUse]
	if !exists {
		return nil, nil
	}

	var toolUses []ToolUse

	// Convert from interface{} to []ToolUse
	actualToolUseBytes, err := json.Marshal(actualToolUseI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal actual tool use: %v", err)
	}

	if err := json.Unmarshal(actualToolUseBytes, &toolUses); err != nil {
		return nil, fmt.Errorf("failed to unmarshal actual tool use: %v", err)
	}

	return toolUses, nil
}

// SetActualToolUse sets the actual tool uses for an evaluation entry
func (e EvaluationEntry) SetActualToolUse(toolUses []ToolUse) {
	e[ActualToolUse] = toolUses
}

// SetResponse sets the response for an evaluation entry
func (e EvaluationEntry) SetResponse(response string) {
	e[Response] = response
}
