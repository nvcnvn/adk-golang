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

package openapi_spec_parser

import (
	"fmt"

	"github.com/nvcnvn/adk-golang/pkg/auth"
	"github.com/nvcnvn/adk-golang/pkg/tools"
	"github.com/nvcnvn/adk-golang/pkg/tools/openapi_tool/common"
)

// RestApiTool represents a tool generated from an OpenAPI specification.
type RestApiTool struct {
	tools.Tool
	name           string
	description    string
	operation      ParsedOperation
	parser         OpenAPISpecParser
	baseURL        string
	authScheme     auth.AuthScheme
	authCredential auth.AuthCredential
}

// NewRestApiTool creates a new RestApiTool.
func NewRestApiTool(
	name string,
	description string,
	operation ParsedOperation,
	parser OpenAPISpecParser,
	baseURL string,
	authScheme auth.AuthScheme,
	authCredential auth.AuthCredential,
) *RestApiTool {
	return &RestApiTool{
		name:           name,
		description:    description,
		operation:      operation,
		parser:         parser,
		baseURL:        baseURL,
		authScheme:     authScheme,
		authCredential: authCredential,
	}
}

// Name returns the name of the tool.
func (t *RestApiTool) Name() string {
	return t.name
}

// Description returns the description of the tool.
func (t *RestApiTool) Description() string {
	return t.description
}

// OpenAPIToolset generates tools from an OpenAPI specification.
type OpenAPIToolset struct {
	specDict       map[string]any
	authScheme     auth.AuthScheme
	authCredential auth.AuthCredential
	baseURL        string
	parser         OpenAPISpecParser
}

// NewOpenAPIToolset creates a new OpenAPIToolset.
func NewOpenAPIToolset(
	specDict map[string]any,
	authScheme auth.AuthScheme,
	authCredential auth.AuthCredential,
) *OpenAPIToolset {
	// Default implementation - in a real implementation, we would determine the actual parser
	// based on the OpenAPI version or other characteristics
	parser := &DefaultOpenAPIParser{}

	return &OpenAPIToolset{
		specDict:       specDict,
		authScheme:     authScheme,
		authCredential: authCredential,
		parser:         parser,
	}
}

// GetTools returns all tools generated from the OpenAPI specification.
func (t *OpenAPIToolset) GetTools() []tools.Tool {
	var result []tools.Tool

	// Parse the OpenAPI specification
	operations, err := t.parser.ParseSpec(t.specDict)
	if err != nil {
		return result
	}

	// Generate a tool for each operation
	for _, op := range operations {
		name := op.ID
		if name == "" {
			name = fmt.Sprintf("%s_%s", op.Endpoint.Method, common.ToSnakeCase(op.Endpoint.Path))
		}

		description := op.Description
		if description == "" {
			description = op.Summary
		}

		tool := NewRestApiTool(
			name,
			description,
			op,
			t.parser,
			t.baseURL,
			t.authScheme,
			t.authCredential,
		)

		result = append(result, tool)
	}

	return result
}

// DefaultOpenAPIParser is a basic implementation of OpenAPISpecParser.
type DefaultOpenAPIParser struct{}

// ParseSpec parses an OpenAPI specification and returns the operations.
func (p *DefaultOpenAPIParser) ParseSpec(spec map[string]any) ([]ParsedOperation, error) {
	// This is a simplified implementation
	// A complete implementation would parse all aspects of an OpenAPI spec
	var operations []ParsedOperation

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		return operations, nil
	}

	for path, pathItem := range paths {
		pathItemMap, ok := pathItem.(map[string]any)
		if !ok {
			continue
		}

		// For each HTTP method in the path
		for method, operation := range pathItemMap {
			// Skip if not an HTTP method
			if method != "get" && method != "put" && method != "post" &&
				method != "delete" && method != "options" &&
				method != "head" && method != "patch" && method != "trace" {
				continue
			}

			opMap, ok := operation.(map[string]any)
			if !ok {
				continue
			}

			op := ParsedOperation{
				Endpoint: OperationEndpoint{
					Path:   path,
					Method: method,
				},
				Parameters:  make(map[string]any),
				RequestBody: make(map[string]any),
				Responses:   make(map[string]any),
			}

			// Extract operation ID, summary, description
			if id, ok := opMap["operationId"].(string); ok {
				op.ID = id
			}

			if summary, ok := opMap["summary"].(string); ok {
				op.Summary = summary
			}

			if description, ok := opMap["description"].(string); ok {
				op.Description = description
			}

			if tags, ok := opMap["tags"].([]any); ok {
				for _, tag := range tags {
					if tagStr, ok := tag.(string); ok {
						op.Tags = append(op.Tags, tagStr)
					}
				}
			}

			// Extract parameters
			if params, ok := opMap["parameters"].([]any); ok {
				for _, param := range params {
					if paramMap, ok := param.(map[string]any); ok {
						if name, ok := paramMap["name"].(string); ok {
							op.Parameters[name] = paramMap
						}
					}
				}
			}

			// Extract request body
			if reqBody, ok := opMap["requestBody"].(map[string]any); ok {
				op.RequestBody = reqBody
			}

			// Extract responses
			if responses, ok := opMap["responses"].(map[string]any); ok {
				op.Responses = responses
			}

			operations = append(operations, op)
		}
	}

	return operations, nil
}
