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
	"fmt"

	"github.com/nvcnvn/adk-golang/pkg/models"
)

// BuiltInCodeExecutionTool is a built-in code execution tool that is automatically
// invoked by Gemini 2 models. This tool operates internally within the model and
// does not require or perform local code execution.
type BuiltInCodeExecutionTool struct {
	*LlmToolAdaptor
}

// NewBuiltInCodeExecutionTool creates a new built-in code execution tool.
func NewBuiltInCodeExecutionTool() *BuiltInCodeExecutionTool {
	// Create a dummy BaseTool since this is a built-in tool that doesn't need execution
	dummyBaseTool := &BaseTool{
		name:        "code_execution",
		description: "Executes code within the model safely",
		schema: ToolSchema{
			Input:  ParameterSchema{Type: "object"},
			Output: map[string]ParameterSchema{},
		},
		executeFn: func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("this is a built-in tool handled by the LLM")
		},
	}

	tool := &BuiltInCodeExecutionTool{
		LlmToolAdaptor: NewLlmToolAdaptor(dummyBaseTool, false),
	}

	tool.SetProcessLlmRequestFunc(tool.processLlmRequest)
	return tool
}

// processLlmRequest modifies the LLM request to include code execution capability.
func (b *BuiltInCodeExecutionTool) processLlmRequest(ctx context.Context, toolContext *ToolContext, llmRequest *models.LlmRequest) error {
	// In the original code, we were checking if the model was a Gemini 2.x model
	// Since we don't have direct access to a model field in LlmRequest,
	// we'll add the code execution tool regardless of model
	// In a real implementation, we would need to get the model name from somewhere

	// Add the code execution tool to the tools list
	codeExecutionTool := &models.Tool{
		Name:          "code_execution",
		Description:   "Executes code within the model safely",
		IsLongRunning: false,
	}

	llmRequest.Tools = append(llmRequest.Tools, codeExecutionTool)

	if llmRequest.ToolsDict == nil {
		llmRequest.ToolsDict = make(map[string]*models.Tool)
	}

	llmRequest.ToolsDict["code_execution"] = codeExecutionTool

	return nil
}

// Create a singleton instance of the built-in code execution tool
var BuiltInCodeExecution = NewBuiltInCodeExecutionTool()
