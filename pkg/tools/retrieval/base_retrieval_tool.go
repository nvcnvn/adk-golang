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

// Package retrieval provides tools for retrieving information from various sources.
package retrieval

import (
	"context"
)

// BaseRetrievalTool provides a common implementation for retrieval tools.
type BaseRetrievalTool struct {
	name        string
	description string
}

// NewBaseRetrievalTool creates a new base retrieval tool.
func NewBaseRetrievalTool(name, description string) *BaseRetrievalTool {
	return &BaseRetrievalTool{
		name:        name,
		description: description,
	}
}

// Name returns the name of the tool.
func (b *BaseRetrievalTool) Name() string {
	return b.name
}

// Description returns a description of what the tool does.
func (b *BaseRetrievalTool) Description() string {
	return b.description
}

// Schema returns the JSON schema for the tool's parameters and return values.
func (b *BaseRetrievalTool) Schema() ToolSchema {
	return ToolSchema{
		Input: ParameterSchema{
			Type:        "object",
			Description: "Input parameters for the retrieval tool",
			Properties: map[string]ParameterSchema{
				"query": {
					Type:        "string",
					Description: "The query to retrieve information for",
					Required:    true,
				},
			},
		},
		Output: map[string]ParameterSchema{
			"result": {
				Type:        "string",
				Description: "The retrieved information",
			},
		},
	}
}

// ToolSchema defines the input/output schema for a tool.
type ToolSchema struct {
	Input  ParameterSchema            `json:"input"`
	Output map[string]ParameterSchema `json:"output"`
}

// ParameterSchema defines the schema for a single parameter.
type ParameterSchema struct {
	Type        string                     `json:"type"`
	Description string                     `json:"description"`
	Required    bool                       `json:"required,omitempty"`
	Properties  map[string]ParameterSchema `json:"properties,omitempty"`
}

// Execute runs the tool with the given input.
// This is a placeholder method that should be overridden by concrete implementations.
func (b *BaseRetrievalTool) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// This should be implemented by derived types
	return map[string]interface{}{
		"error": "BaseRetrievalTool.Execute must be implemented by derived types",
	}, nil
}
