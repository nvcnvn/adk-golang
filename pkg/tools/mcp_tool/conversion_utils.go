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

package mcp_tool

import (
	"github.com/nvcnvn/adk-golang/pkg/tools"
)

// AdkToMcpToolType converts an ADK tool to an MCP tool type.
func AdkToMcpToolType(tool tools.Tool) (McpBaseTool, error) {
	// Convert the tool schema to a JSON schema
	schema := tool.Schema()
	inputSchema := convertFromParameterSchema(schema.Input)

	return &genericMcpTool{
		name:        tool.Name(),
		description: tool.Description(),
		inputSchema: inputSchema,
	}, nil
}

// genericMcpTool is a simple implementation of the McpBaseTool interface.
type genericMcpTool struct {
	name        string
	description string
	inputSchema map[string]interface{}
}

func (t *genericMcpTool) Name() string {
	return t.name
}

func (t *genericMcpTool) Description() string {
	return t.description
}

func (t *genericMcpTool) InputSchema() map[string]interface{} {
	return t.inputSchema
}

// convertFromParameterSchema converts a ParameterSchema to a JSON schema.
func convertFromParameterSchema(paramSchema tools.ParameterSchema) map[string]interface{} {
	schemaMap := make(map[string]interface{})
	schemaMap["type"] = paramSchema.Type

	if paramSchema.Description != "" {
		schemaMap["description"] = paramSchema.Description
	}

	if len(paramSchema.Properties) > 0 {
		props := make(map[string]interface{})
		required := make([]string, 0)

		for name, propSchema := range paramSchema.Properties {
			props[name] = convertFromParameterSchema(propSchema)

			if propSchema.Required {
				required = append(required, name)
			}
		}

		schemaMap["properties"] = props

		if len(required) > 0 {
			schemaMap["required"] = required
		}
	}

	return schemaMap
}
