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

// Package tools provides interfaces and implementations for various tools that agents can use.
package tools

import (
	"context"
)

// Tool represents a capability that can be provided to an agent.
type Tool interface {
	// Name returns the name of the tool.
	Name() string

	// Description returns a description of what the tool does.
	Description() string

	// Execute runs the tool with the given input.
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

	// Schema returns the JSON schema for the tool's parameters and return values.
	Schema() ToolSchema
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

// BaseTool provides a common implementation of the Tool interface.
type BaseTool struct {
	name        string
	description string
	schema      ToolSchema
	executeFn   func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

// NewTool creates a new tool with the given name, description, schema, and execute function.
func NewTool(
	name, description string,
	schema ToolSchema,
	executeFn func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error),
) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		schema:      schema,
		executeFn:   executeFn,
	}
}

// Name returns the name of the tool.
func (b *BaseTool) Name() string {
	return b.name
}

// Description returns a description of what the tool does.
func (b *BaseTool) Description() string {
	return b.description
}

// Execute runs the tool with the given input.
func (b *BaseTool) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return b.executeFn(ctx, input)
}

// Schema returns the JSON schema for the tool's parameters and return values.
func (b *BaseTool) Schema() ToolSchema {
	return b.schema
}
