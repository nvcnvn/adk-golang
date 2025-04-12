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

// Package code_executors provides functionality for executing code snippets.
package code_executors

import (
	"time"
)

// Key constants for the session state
const (
	contextKey              = "_code_execution_context"
	sessionIDKey            = "execution_session_id"
	processedFileNamesKey   = "processed_input_files"
	inputFileKey            = "_code_executor_input_files"
	errorCountKey           = "_code_executor_error_counts"
	codeExecutionResultsKey = "_code_execution_results"
)

// SessionState represents the interface required for a session state
type SessionState interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}

// CodeExecutionResult represents the result of a code execution
type CodeExecutionResult struct {
	Code         string `json:"code"`
	ResultStdout string `json:"result_stdout"`
	ResultStderr string `json:"result_stderr"`
	Timestamp    int64  `json:"timestamp"`
}

// CodeExecutorContext maintains state between code executions
type CodeExecutorContext struct {
	context      map[string]interface{}
	sessionState SessionState
}

// NewCodeExecutorContext creates a new code executor context
func NewCodeExecutorContext(sessionState SessionState) *CodeExecutorContext {
	ctx := &CodeExecutorContext{
		sessionState: sessionState,
	}
	ctx.context = ctx.getCodeExecutorContext(sessionState)
	return ctx
}

// GetStateDelta returns the state delta to update in the persistent session state
func (c *CodeExecutorContext) GetStateDelta() map[string]interface{} {
	// Deep copy the context
	contextCopy := make(map[string]interface{})
	for k, v := range c.context {
		contextCopy[k] = v
	}
	return map[string]interface{}{
		contextKey: contextCopy,
	}
}

// GetExecutionID gets the session ID for the code executor
func (c *CodeExecutorContext) GetExecutionID() string {
	if sessionID, ok := c.context[sessionIDKey].(string); ok {
		return sessionID
	}
	return ""
}

// SetExecutionID sets the session ID for the code executor
func (c *CodeExecutorContext) SetExecutionID(sessionID string) {
	c.context[sessionIDKey] = sessionID
}

// GetProcessedFileNames gets the processed file names from the session state
func (c *CodeExecutorContext) GetProcessedFileNames() []string {
	if fileNames, ok := c.context[processedFileNamesKey].([]interface{}); ok {
		result := make([]string, len(fileNames))
		for i, name := range fileNames {
			if str, ok := name.(string); ok {
				result[i] = str
			}
		}
		return result
	}
	return []string{}
}

// AddProcessedFileNames adds file names to the list of processed files
func (c *CodeExecutorContext) AddProcessedFileNames(fileNames []string) {
	existing := c.GetProcessedFileNames()
	combined := append(existing, fileNames...)
	c.context[processedFileNamesKey] = combined
}

// GetInputFiles gets the input files from the session state
func (c *CodeExecutorContext) GetInputFiles() []File {
	value, ok := c.sessionState.Get(inputFileKey)
	if !ok {
		return []File{}
	}

	fileData, ok := value.([]interface{})
	if !ok {
		return []File{}
	}

	files := make([]File, 0, len(fileData))
	for _, fileItem := range fileData {
		if fileMap, ok := fileItem.(map[string]interface{}); ok {
			name, _ := fileMap["Name"].(string)
			contentRaw, _ := fileMap["Content"].([]interface{})

			content := make([]byte, len(contentRaw))
			for i, b := range contentRaw {
				if v, ok := b.(float64); ok {
					content[i] = byte(v)
				}
			}

			files = append(files, File{
				Name:    name,
				Content: content,
			})
		}
	}
	return files
}

// AddInputFiles adds input files to the session state
func (c *CodeExecutorContext) AddInputFiles(inputFiles []File) {
	var existingFiles []interface{}

	value, ok := c.sessionState.Get(inputFileKey)
	if ok {
		existingFiles, _ = value.([]interface{})
	} else {
		existingFiles = []interface{}{}
	}

	for _, file := range inputFiles {
		existingFiles = append(existingFiles, map[string]interface{}{
			"Name":    file.Name,
			"Content": file.Content,
		})
	}

	c.sessionState.Set(inputFileKey, existingFiles)
}

// ClearInputFiles removes the input files and processed file names
func (c *CodeExecutorContext) ClearInputFiles() {
	c.sessionState.Set(inputFileKey, []interface{}{})
	if _, ok := c.context[processedFileNamesKey]; ok {
		c.context[processedFileNamesKey] = []string{}
	}
}

// GetErrorCount gets the error count for an invocation ID
func (c *CodeExecutorContext) GetErrorCount(invocationID string) int {
	value, ok := c.sessionState.Get(errorCountKey)
	if !ok {
		return 0
	}

	errorCounts, ok := value.(map[string]interface{})
	if !ok {
		return 0
	}

	count, ok := errorCounts[invocationID]
	if !ok {
		return 0
	}

	countInt, ok := count.(float64)
	if !ok {
		return 0
	}

	return int(countInt)
}

// IncrementErrorCount increments the error count for the given invocation ID
func (c *CodeExecutorContext) IncrementErrorCount(invocationID string) {
	value, ok := c.sessionState.Get(errorCountKey)

	var errorCounts map[string]interface{}
	if ok {
		errorCounts, _ = value.(map[string]interface{})
	}

	if errorCounts == nil {
		errorCounts = make(map[string]interface{})
	}

	errorCounts[invocationID] = c.GetErrorCount(invocationID) + 1
	c.sessionState.Set(errorCountKey, errorCounts)
}

// ResetErrorCount resets the error count for the given invocation ID
func (c *CodeExecutorContext) ResetErrorCount(invocationID string) {
	value, ok := c.sessionState.Get(errorCountKey)
	if !ok {
		return
	}

	errorCounts, ok := value.(map[string]interface{})
	if !ok {
		return
	}

	delete(errorCounts, invocationID)
	c.sessionState.Set(errorCountKey, errorCounts)
}

// UpdateCodeExecutionResult updates the code execution result
func (c *CodeExecutorContext) UpdateCodeExecutionResult(
	invocationID string,
	code string,
	resultStdout string,
	resultStderr string,
) {
	value, ok := c.sessionState.Get(codeExecutionResultsKey)

	var results map[string]interface{}
	if ok {
		results, _ = value.(map[string]interface{})
	}

	if results == nil {
		results = make(map[string]interface{})
	}

	var invocationResults []interface{}
	if invResults, ok := results[invocationID]; ok {
		invocationResults, _ = invResults.([]interface{})
	}

	if invocationResults == nil {
		invocationResults = []interface{}{}
	}

	invocationResults = append(invocationResults, map[string]interface{}{
		"code":          code,
		"result_stdout": resultStdout,
		"result_stderr": resultStderr,
		"timestamp":     time.Now().Unix(),
	})

	results[invocationID] = invocationResults
	c.sessionState.Set(codeExecutionResultsKey, results)
}

// getCodeExecutorContext gets the code executor context from the session state
func (c *CodeExecutorContext) getCodeExecutorContext(sessionState SessionState) map[string]interface{} {
	value, ok := sessionState.Get(contextKey)
	if !ok {
		context := make(map[string]interface{})
		sessionState.Set(contextKey, context)
		return context
	}

	context, ok := value.(map[string]interface{})
	if !ok {
		context = make(map[string]interface{})
		sessionState.Set(contextKey, context)
		return context
	}

	return context
}
