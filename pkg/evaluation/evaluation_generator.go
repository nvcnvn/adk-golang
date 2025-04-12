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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// FunctionCall represents a tool/function call
type FunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// ContentPart represents a part of a content message
type ContentPart struct {
	Text         string        `json:"text,omitempty"`
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
}

// Content represents the content of a message
type Content struct {
	Role  string        `json:"role"`
	Parts []ContentPart `json:"parts"`
}

// Event represents an interaction event
type Event struct {
	Author       string   `json:"author"`
	Content      *Content `json:"content"`
	InvocationID string   `json:"invocation_id"`
}

// Session represents a conversation session
type Session struct {
	Events []Event `json:"events"`
}

// Agent interface defines the minimal requirements for an agent in the evaluation framework
type Agent interface {
	// Any methods needed would be defined here
}

// Tool represents a tool that can be used by an agent
type Tool struct {
	Name string
}

// BeforeToolCallback is a function called before a tool is executed
type BeforeToolCallback func(tool *Tool, args map[string]interface{}, context interface{}, data EvaluationConversation) (map[string]interface{}, error)

// RunnerEvent represents an event emitted by the runner during execution
type RunnerEvent struct {
	content *Content
}

// IsFinalResponse returns true if this event is the final response
func (e *RunnerEvent) IsFinalResponse() bool {
	// This is a simplified implementation
	return e.content != nil && e.content.Role != "user"
}

// GetFunctionCalls returns function calls from the event
func (e *RunnerEvent) GetFunctionCalls() []FunctionCall {
	if e.content == nil {
		return nil
	}

	var calls []FunctionCall
	for _, part := range e.content.Parts {
		if part.FunctionCall != nil {
			calls = append(calls, *part.FunctionCall)
		}
	}

	return calls
}

// Runner interface defines the minimum functionality for a runner
type Runner interface {
	Run(userID string, sessionID string, content map[string]interface{}) ([]*RunnerEvent, error)
}

// EvaluationGenerator generates evaluation responses for agents
type EvaluationGenerator struct{}

// NewEvaluationGenerator creates a new evaluation generator
func NewEvaluationGenerator() *EvaluationGenerator {
	return &EvaluationGenerator{}
}

// GenerateResponses returns evaluation responses for the given dataset and agent
func (eg *EvaluationGenerator) GenerateResponses(
	evalDataset EvaluationDataset,
	rootAgent Agent,
	repeatNum int,
	agentName string,
	initialSession map[string]interface{},
) (EvaluationDataset, error) {
	// This is a skeleton implementation
	// In a real implementation, this would interact with the agent to generate responses
	fmt.Println("Note: GenerateResponses is implemented as a skeleton. The actual functionality requires integration with agent modules.")

	// For now, we'll return the input dataset as is
	return evalDataset, nil
}

// GenerateResponsesFromSession returns evaluation responses by combining session data with eval data
func (eg *EvaluationGenerator) GenerateResponsesFromSession(
	sessionPath string,
	evalDataset EvaluationDataset,
) (EvaluationDataset, error) {
	// Read session data from file
	sessionData, err := ioutil.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %v", err)
	}

	// Parse session data
	var session Session
	if err := json.Unmarshal(sessionData, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session data: %v", err)
	}

	fmt.Printf("Loaded session: %s\n", sessionPath)

	results := make(EvaluationDataset, 0, len(evalDataset))

	for _, data := range evalDataset {
		processedData, err := eg.ProcessQueryWithSession(session, data)
		if err != nil {
			return nil, err
		}
		results = append(results, processedData)
	}

	return results, nil
}

// ProcessQueryWithAgent processes a query using the agent and evaluation dataset
func (eg *EvaluationGenerator) ProcessQueryWithAgent(
	data EvaluationConversation,
	rootAgent Agent,
	agentName string,
	initialSession map[string]interface{},
) (EvaluationConversation, error) {
	// This is a skeleton implementation
	// In a real implementation, this would use the agent to process queries and gather responses
	fmt.Println("Note: ProcessQueryWithAgent is implemented as a skeleton. The actual functionality requires integration with agent modules.")

	// For now, we'll return the input data as is
	return data, nil
}

// ProcessQueryWithSession processes queries using existing session data without invoking the runner
func (eg *EvaluationGenerator) ProcessQueryWithSession(
	sessionData Session,
	data EvaluationConversation,
) (EvaluationConversation, error) {
	responses := make(EvaluationConversation, len(data))
	copy(responses, data)

	// Iterate through the provided queries and align them with the session events
	for i, entry := range responses {
		query, ok := entry[Query].(string)
		if !ok {
			return nil, fmt.Errorf("invalid query in entry %d", i)
		}

		var actualToolUses []ToolUse
		var response string

		// Search for the corresponding session events
		for _, event := range sessionData.Events {
			// Match the query to a user event
			if event.Author == "user" &&
				event.Content != nil &&
				len(event.Content.Parts) > 0 &&
				event.Content.Parts[0].Text == query {

				// Look for subsequent tool usage or model responses for this invocation
				for _, subsequent := range sessionData.Events {
					if subsequent.InvocationID == event.InvocationID {
						// Extract tool usage
						if len(subsequent.Content.Parts) > 0 && subsequent.Content.Parts[0].FunctionCall != nil {
							call := subsequent.Content.Parts[0].FunctionCall
							toolUse := ToolUse{
								ToolName:  call.Name,
								ToolInput: call.Args,
							}
							actualToolUses = append(actualToolUses, toolUse)
						} else if subsequent.Author != "user" && len(subsequent.Content.Parts) > 0 {
							// Extract final response (from non-user)
							response = subsequent.Content.Parts[0].Text
						}
					}
				}
			}
		}

		// Update the results for the current query
		responses[i][ActualToolUse] = actualToolUses
		responses[i][Response] = response
	}

	return responses, nil
}

// beforeToolCallback intercepts specific tool calls and returns predefined outputs from eval_dataset
func (eg *EvaluationGenerator) beforeToolCallback(
	tool *Tool,
	args map[string]interface{},
	toolContext interface{},
	evalDataset EvaluationConversation,
) (map[string]interface{}, error) {
	for i, entry := range evalDataset {
		expectedToolUseI, exists := entry[ExpectedToolUse]
		if !exists {
			continue
		}

		// Convert to []interface{}
		expectedToolUses, ok := expectedToolUseI.([]interface{})
		if !ok {
			continue
		}

		// Look for matching tool use
		for _, expectedToolUseI := range expectedToolUses {
			expectedToolUse, ok := expectedToolUseI.(map[string]interface{})
			if !ok {
				continue
			}

			mockToolOutputI, hasMockOutput := expectedToolUse[MockToolOutput]
			if !hasMockOutput {
				continue
			}

			toolNameI, hasToolName := expectedToolUse[ToolName]
			if !hasToolName {
				continue
			}

			toolName, ok := toolNameI.(string)
			if !ok || toolName != tool.Name {
				continue
			}

			toolInputI, hasToolInput := expectedToolUse[ToolInput]
			if !hasToolInput {
				continue
			}

			toolInput, ok := toolInputI.(map[string]interface{})
			if !ok {
				continue
			}

			// Check if the tool input matches the args
			if reflect.DeepEqual(toolInput, args) {
				// Remove the matched entry so we don't rematch again
				// This requires modifying the dataset which may have side effects
				evalDataset = append(evalDataset[:i], evalDataset[i+1:]...)
				return map[string]interface{}{"result": mockToolOutputI}, nil
			}
		}
	}

	return nil, nil
}

// applyBeforeToolCallback applies the before_tool_callback to agents with matching tools
func (eg *EvaluationGenerator) applyBeforeToolCallback(
	agent Agent,
	callback BeforeToolCallback,
	mockToolNames map[string]bool,
) {
	// This is a placeholder implementation
	fmt.Println("Note: Tool callback functionality would need implementation based on agent structure")
}

// LoadDataset loads evaluation data from file paths or directories
func (eg *EvaluationGenerator) LoadDataset(inputData interface{}) (EvaluationDataset, error) {
	dataset := EvaluationDataset{}

	switch data := inputData.(type) {
	case string:
		// Handle single string path (file or directory)
		fileInfo, err := os.Stat(data)
		if err != nil {
			return nil, fmt.Errorf("invalid path %s: %v", data, err)
		}

		if fileInfo.IsDir() {
			// Process directory
			testFiles, err := eg.findTestFilesInDir(data)
			if err != nil {
				return nil, err
			}

			for _, filePath := range testFiles {
				conversation, err := eg.loadJSONFile(filePath)
				if err != nil {
					return nil, err
				}
				dataset = append(dataset, conversation)
			}
		} else {
			// Process single file
			conversation, err := eg.loadJSONFile(data)
			if err != nil {
				return nil, err
			}
			dataset = append(dataset, conversation)
		}

	case []string:
		// Handle slice of file paths
		for _, filePath := range data {
			conversation, err := eg.loadJSONFile(filePath)
			if err != nil {
				return nil, err
			}
			dataset = append(dataset, conversation)
		}

	default:
		return nil, fmt.Errorf("unsupported input type for dataset loading")
	}

	return dataset, nil
}

// findTestFilesInDir finds all .test.json files in a directory and its subdirectories
func (eg *EvaluationGenerator) findTestFilesInDir(dirPath string) ([]string, error) {
	var testFiles []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".json" && strings.HasSuffix(info.Name(), ".test.json") {
			testFiles = append(testFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %v", dirPath, err)
	}

	return testFiles, nil
}

// loadJSONFile loads a JSON file containing evaluation data
func (eg *EvaluationGenerator) loadJSONFile(filePath string) (EvaluationConversation, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
	}

	var conversation EvaluationConversation
	if err := json.Unmarshal(data, &conversation); err != nil {
		return nil, fmt.Errorf("failed to parse JSON in file %s: %v", filePath, err)
	}

	// Validate the conversation data
	for i, entry := range conversation {
		if _, exists := entry[Query]; !exists {
			return nil, fmt.Errorf("entry %d in file %s is missing required 'query' field", i, filePath)
		}
	}

	return conversation, nil
}
