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

// Package mcp_tool provides functionality for interacting with MCP (Model Context Protocol) tools.
package mcp_tool

import (
	"context"
	"errors"
	"fmt"

	"github.com/nvcnvn/adk-golang/pkg/auth"
	"github.com/nvcnvn/adk-golang/pkg/tools"
)

// McpTool represents a tool that can be executed through an MCP session.
type McpTool struct {
	name           string
	description    string
	mcpTool        McpBaseTool
	mcpSession     ClientSession
	authScheme     auth.AuthScheme
	authCredential auth.AuthCredential
	schema         tools.ToolSchema
}

// McpBaseTool defines the interface for an MCP tool.
type McpBaseTool interface {
	// Name returns the name of the tool.
	Name() string
	// Description returns the description of the tool.
	Description() string
	// InputSchema returns the input schema for the tool.
	InputSchema() map[string]interface{}
}

// ClientSession defines the interface for interacting with an MCP session.
type ClientSession interface {
	// CallTool calls the named tool with the provided arguments.
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error)
	// Initialize initializes the session.
	Initialize(ctx context.Context) error
	// ListTools lists all available tools.
	ListTools(ctx context.Context) (*ListToolsResult, error)
}

// ListToolsResult represents the result of a list tools operation.
type ListToolsResult struct {
	Tools []McpBaseTool
}

// NewMcpTool creates a new MCP tool.
func NewMcpTool(
	mcpTool McpBaseTool,
	mcpSession ClientSession,
	authScheme auth.AuthScheme,
	authCredential *auth.AuthCredential,
) (*McpTool, error) {
	if mcpTool == nil {
		return nil, errors.New("mcpTool cannot be nil")
	}
	if mcpSession == nil {
		return nil, errors.New("mcpSession cannot be nil")
	}

	// Convert input schema to tool schema
	schema := tools.ToolSchema{
		Input: convertToParameterSchema(mcpTool.InputSchema()),
		// Note: We don't have output schema information from MCP tools
		Output: map[string]tools.ParameterSchema{
			"result": {
				Type:        "object",
				Description: "The result of the tool execution",
			},
		},
	}

	return &McpTool{
		name:           mcpTool.Name(),
		description:    mcpTool.Description(),
		mcpTool:        mcpTool,
		mcpSession:     mcpSession,
		authScheme:     authScheme,
		authCredential: *authCredential,
		schema:         schema,
	}, nil
}

// convertToParameterSchema converts a JSON schema to a ParameterSchema.
func convertToParameterSchema(schema map[string]interface{}) tools.ParameterSchema {
	paramSchema := tools.ParameterSchema{
		Type:        "object",
		Description: "The parameters for the tool",
	}

	properties, hasProps := schema["properties"].(map[string]interface{})
	if hasProps {
		paramProps := make(map[string]tools.ParameterSchema)
		for name, propSchema := range properties {
			if propMap, ok := propSchema.(map[string]interface{}); ok {
				propType := "string"
				if t, ok := propMap["type"].(string); ok {
					propType = t
				}

				propDesc := ""
				if d, ok := propMap["description"].(string); ok {
					propDesc = d
				}

				required := false
				if req, ok := schema["required"].([]interface{}); ok {
					for _, reqName := range req {
						if reqStr, ok := reqName.(string); ok && reqStr == name {
							required = true
							break
						}
					}
				}

				paramProps[name] = tools.ParameterSchema{
					Type:        propType,
					Description: propDesc,
					Required:    required,
				}
			}
		}
		paramSchema.Properties = paramProps
	}

	return paramSchema
}

// Name returns the name of the tool.
func (mt *McpTool) Name() string {
	return mt.name
}

// Description returns the description of the tool.
func (mt *McpTool) Description() string {
	return mt.description
}

// Schema returns the schema for the tool.
func (mt *McpTool) Schema() tools.ToolSchema {
	return mt.schema
}

// Execute runs the tool with the given arguments.
func (mt *McpTool) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// TODO: Support passing auth to MCP Server.
	response, err := mt.mcpSession.CallTool(ctx, mt.name, input)
	if err != nil {
		return nil, fmt.Errorf("failed to execute MCP tool: %v", err)
	}

	// If the response is already a map[string]interface{}, return it
	if responseMap, ok := response.(map[string]interface{}); ok {
		return responseMap, nil
	}

	// Otherwise, wrap it in a map
	return map[string]interface{}{
		"result": response,
	}, nil
}

// EnsureToolsInterface ensures McpTool implements tools.Tool.
var _ tools.Tool = (*McpTool)(nil)
