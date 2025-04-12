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

// VertexAISearchTool is a built-in tool that integrates with Vertex AI Search.
type VertexAISearchTool struct {
	*LlmToolAdaptor
	dataStoreID    string
	searchEngineID string
}

// NewVertexAISearchTool creates a new Vertex AI Search tool.
// It requires either a dataStoreID or a searchEngineID, but not both.
func NewVertexAISearchTool(dataStoreID, searchEngineID string) (*VertexAISearchTool, error) {
	// Either dataStoreID or searchEngineID must be specified, but not both
	if (dataStoreID == "" && searchEngineID == "") || (dataStoreID != "" && searchEngineID != "") {
		return nil, fmt.Errorf("either dataStoreID or searchEngineID must be specified, but not both")
	}

	// Create a dummy BaseTool since this is a built-in tool that doesn't need execution
	dummyBaseTool := &BaseTool{
		name:        "vertex_ai_search",
		description: "Search using Vertex AI Search",
		schema: ToolSchema{
			Input:  ParameterSchema{Type: "object"},
			Output: map[string]ParameterSchema{},
		},
		executeFn: func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("this is a built-in tool handled by the LLM")
		},
	}

	tool := &VertexAISearchTool{
		LlmToolAdaptor: NewLlmToolAdaptor(dummyBaseTool, false),
		dataStoreID:    dataStoreID,
		searchEngineID: searchEngineID,
	}

	tool.SetProcessLlmRequestFunc(tool.processLlmRequest)
	return tool, nil
}

// processLlmRequest modifies the LLM request to include Vertex AI Search configuration.
func (v *VertexAISearchTool) processLlmRequest(ctx context.Context, toolContext *ToolContext, llmRequest *models.LlmRequest) error {
	// We need the model name to check if it's a Gemini model
	// This depends on implementation details of how the model name is stored/accessed
	// Since we don't have access to a direct Model field, we'll need to infer it
	// from other sources (e.g., system instructions, context, etc.)

	// For this implementation, we'll assume Gemini models are being used
	// In a real implementation, we would need to get the model name from somewhere

	// Add the Vertex AI Search configuration to the tools list
	vertexAISearchTool := &models.Tool{
		Name:          "vertex_ai_search",
		Description:   "Search using Vertex AI Search",
		IsLongRunning: false,
	}

	// Set metadata in ToolsDict
	metadata := make(map[string]interface{})
	if v.dataStoreID != "" {
		metadata["datastore"] = v.dataStoreID
	}
	if v.searchEngineID != "" {
		metadata["engine"] = v.searchEngineID
	}
	if llmRequest.ToolsDict == nil {
		llmRequest.ToolsDict = make(map[string]*models.Tool)
	}
	vertexAISearchTool.InputSchema = metadata
	llmRequest.Tools = append(llmRequest.Tools, vertexAISearchTool)
	llmRequest.ToolsDict["vertex_ai_search"] = vertexAISearchTool

	return nil
}

// VertexAISearchWithDataStore creates a new Vertex AI Search tool with the given data store ID.
// This is a convenience function that creates a new VertexAISearchTool with a data store ID.
func VertexAISearchWithDataStore(dataStoreID string) (*VertexAISearchTool, error) {
	return NewVertexAISearchTool(dataStoreID, "")
}

// VertexAISearchWithEngine creates a new Vertex AI Search tool with the given search engine ID.
// This is a convenience function that creates a new VertexAISearchTool with a search engine ID.
func VertexAISearchWithEngine(searchEngineID string) (*VertexAISearchTool, error) {
	return NewVertexAISearchTool("", searchEngineID)
}
