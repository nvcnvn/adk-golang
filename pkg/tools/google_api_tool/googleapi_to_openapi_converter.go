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

package googleapitool

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// GoogleApiToOpenApiConverter converts Google API Discovery documents to OpenAPI v3 format.
type GoogleApiToOpenApiConverter struct {
	apiName      string
	apiVersion   string
	googleApiDoc map[string]interface{}
	openApiSpec  map[string]interface{}
}

// NewGoogleApiToOpenApiConverter creates a new converter for the specified Google API.
func NewGoogleApiToOpenApiConverter(apiName, apiVersion string) *GoogleApiToOpenApiConverter {
	return &GoogleApiToOpenApiConverter{
		apiName:    apiName,
		apiVersion: apiVersion,
		openApiSpec: map[string]interface{}{
			"openapi": "3.0.0",
			"info":    map[string]interface{}{},
			"servers": []interface{}{},
			"paths":   map[string]interface{}{},
			"components": map[string]interface{}{
				"schemas":         map[string]interface{}{},
				"securitySchemes": map[string]interface{}{},
			},
		},
	}
}

// fetchGoogleApiSpec fetches the Google API specification using the discovery service.
func (c *GoogleApiToOpenApiConverter) fetchGoogleApiSpec() error {
	log.Printf("Fetching Google API spec for %s %s", c.apiName, c.apiVersion)

	// Construct the discovery URL for the API
	discoveryURL := fmt.Sprintf(
		"https://discovery.googleapis.com/discovery/v1/apis/%s/%s/rest",
		c.apiName, c.apiVersion,
	)

	// Make an HTTP request to fetch the API description
	resp, err := http.Get(discoveryURL)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON
	err = json.Unmarshal(body, &c.googleApiDoc)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	if c.googleApiDoc == nil {
		return fmt.Errorf("failed to retrieve API specification")
	}

	log.Printf("Successfully fetched %s API specification", c.apiName)
	return nil
}

// Convert processes the Google API spec and returns an OpenAPI v3 spec.
func (c *GoogleApiToOpenApiConverter) Convert() (map[string]interface{}, error) {
	if c.googleApiDoc == nil {
		if err := c.fetchGoogleApiSpec(); err != nil {
			return nil, err
		}
	}

	// Convert basic API information
	c.convertInfo()

	// Convert server information
	c.convertServers()

	// Convert authentication schemes
	c.convertSecuritySchemes()

	// Convert schemas (models)
	c.convertSchemas()

	// Convert endpoints/paths
	if resources, ok := c.googleApiDoc["resources"].(map[string]interface{}); ok {
		c.convertResources(resources, "")
	}

	// Convert top-level methods
	if methods, ok := c.googleApiDoc["methods"].(map[string]interface{}); ok {
		c.convertMethods(methods, "/")
	}

	return c.openApiSpec, nil
}

// convertInfo converts basic API information.
func (c *GoogleApiToOpenApiConverter) convertInfo() {
	title := c.apiName + " API"
	if t, ok := c.googleApiDoc["title"].(string); ok && t != "" {
		title = t
	}

	desc := ""
	if d, ok := c.googleApiDoc["description"].(string); ok {
		desc = d
	}

	version := c.apiVersion
	if v, ok := c.googleApiDoc["version"].(string); ok && v != "" {
		version = v
	}

	c.openApiSpec["info"] = map[string]interface{}{
		"title":       title,
		"description": desc,
		"version":     version,
		"contact":     map[string]interface{}{},
	}

	if docsLink, ok := c.googleApiDoc["documentationLink"].(string); ok && docsLink != "" {
		c.openApiSpec["info"].(map[string]interface{})["termsOfService"] = docsLink
		c.openApiSpec["externalDocs"] = map[string]interface{}{
			"description": "API Documentation",
			"url":         docsLink,
		}
	}
}

// convertServers converts server information.
func (c *GoogleApiToOpenApiConverter) convertServers() {
	rootURL := ""
	if url, ok := c.googleApiDoc["rootUrl"].(string); ok {
		rootURL = url
	}

	servicePath := ""
	if path, ok := c.googleApiDoc["servicePath"].(string); ok {
		servicePath = path
	}

	baseURL := rootURL + servicePath
	if strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL[:len(baseURL)-1]
	}

	c.openApiSpec["servers"] = []interface{}{
		map[string]interface{}{
			"url":         baseURL,
			"description": fmt.Sprintf("%s %s API", c.apiName, c.apiVersion),
		},
	}
}

// convertSecuritySchemes converts authentication and authorization schemes.
func (c *GoogleApiToOpenApiConverter) convertSecuritySchemes() {
	securitySchemes := map[string]interface{}{}
	security := []interface{}{}

	// Check if there's OAuth2 configuration
	if auth, ok := c.googleApiDoc["auth"].(map[string]interface{}); ok {
		if oauth2, ok := auth["oauth2"].(map[string]interface{}); ok {
			if scopes, ok := oauth2["scopes"].(map[string]interface{}); ok && len(scopes) > 0 {
				formattedScopes := map[string]interface{}{}

				for scope, scopeInfo := range scopes {
					if scopeInfoMap, ok := scopeInfo.(map[string]interface{}); ok {
						if desc, ok := scopeInfoMap["description"].(string); ok {
							formattedScopes[scope] = desc
						} else {
							formattedScopes[scope] = ""
						}
					}
				}

				// Add OAuth2 security scheme
				securitySchemes["oauth2"] = map[string]interface{}{
					"type":        "oauth2",
					"description": "OAuth 2.0 authentication",
					"flows": map[string]interface{}{
						"authorizationCode": map[string]interface{}{
							"authorizationUrl": "https://accounts.google.com/o/oauth2/auth",
							"tokenUrl":         "https://oauth2.googleapis.com/token",
							"scopes":           formattedScopes,
						},
					},
				}

				// Add OAuth2 to the global security requirement
				scopeKeys := []string{}
				for key := range formattedScopes {
					scopeKeys = append(scopeKeys, key)
				}
				security = append(security, map[string]interface{}{
					"oauth2": scopeKeys,
				})
			}
		}
	}

	// Add API key authentication (most Google APIs support this)
	securitySchemes["apiKey"] = map[string]interface{}{
		"type":        "apiKey",
		"in":          "query",
		"name":        "key",
		"description": "API key for accessing this API",
	}
	security = append(security, map[string]interface{}{"apiKey": []string{}})

	// Set the security schemes
	c.openApiSpec["components"].(map[string]interface{})["securitySchemes"] = securitySchemes
	c.openApiSpec["security"] = security
}

// convertSchemas converts schema definitions.
func (c *GoogleApiToOpenApiConverter) convertSchemas() {
	schemas := map[string]interface{}{}

	if googleSchemas, ok := c.googleApiDoc["schemas"].(map[string]interface{}); ok {
		for schemaName, schemaDef := range googleSchemas {
			if schemaDefMap, ok := schemaDef.(map[string]interface{}); ok {
				schemas[schemaName] = c.convertSchemaObject(schemaDefMap)
			}
		}
	}

	c.openApiSpec["components"].(map[string]interface{})["schemas"] = schemas
}

// convertSchemaObject recursively converts a Google API schema object to OpenAPI schema.
func (c *GoogleApiToOpenApiConverter) convertSchemaObject(schemaDef map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}

	// Convert the type
	if typeName, ok := schemaDef["type"].(string); ok {
		if typeName == "object" {
			result["type"] = "object"

			// Handle properties
			if props, ok := schemaDef["properties"].(map[string]interface{}); ok {
				properties := map[string]interface{}{}
				requiredFields := []string{}

				for propName, propDef := range props {
					if propDefMap, ok := propDef.(map[string]interface{}); ok {
						properties[propName] = c.convertSchemaObject(propDefMap)

						// Check if property is required
						if req, ok := propDefMap["required"].(bool); ok && req {
							requiredFields = append(requiredFields, propName)
						}
					}
				}

				result["properties"] = properties
				if len(requiredFields) > 0 {
					result["required"] = requiredFields
				}
			}

		} else if typeName == "array" {
			result["type"] = "array"
			if items, ok := schemaDef["items"].(map[string]interface{}); ok {
				result["items"] = c.convertSchemaObject(items)
			}

		} else if typeName == "any" {
			// OpenAPI doesn't have direct "any" type, use oneOf as alternative
			result["oneOf"] = []interface{}{
				map[string]interface{}{"type": "object"},
				map[string]interface{}{"type": "array"},
				map[string]interface{}{"type": "string"},
				map[string]interface{}{"type": "number"},
				map[string]interface{}{"type": "boolean"},
				map[string]interface{}{"type": "null"},
			}
		} else {
			result["type"] = typeName
		}
	}

	// Handle references
	if ref, ok := schemaDef["$ref"].(string); ok {
		if strings.HasPrefix(ref, "#") {
			result["$ref"] = strings.Replace(ref, "#", "#/components/schemas/", 1)
		} else {
			result["$ref"] = "#/components/schemas/" + ref
		}
	}

	// Handle format
	if format, ok := schemaDef["format"].(string); ok {
		result["format"] = format
	}

	// Handle enum values
	if enum, ok := schemaDef["enum"].([]interface{}); ok {
		result["enum"] = enum
	}

	// Handle description
	if desc, ok := schemaDef["description"].(string); ok {
		result["description"] = desc
	}

	// Handle pattern
	if pattern, ok := schemaDef["pattern"].(string); ok {
		result["pattern"] = pattern
	}

	// Handle default value
	if defaultVal, ok := schemaDef["default"]; ok {
		result["default"] = defaultVal
	}

	return result
}

// convertResources recursively converts all resources and their methods.
func (c *GoogleApiToOpenApiConverter) convertResources(resources map[string]interface{}, parentPath string) {
	for resourceName, resourceData := range resources {
		if resourceDataMap, ok := resourceData.(map[string]interface{}); ok {
			resourcePath := fmt.Sprintf("%s/%s", parentPath, resourceName)

			// Process methods for this resource
			if methods, ok := resourceDataMap["methods"].(map[string]interface{}); ok {
				c.convertMethods(methods, resourcePath)
			}

			// Process nested resources recursively
			if nestedResources, ok := resourceDataMap["resources"].(map[string]interface{}); ok {
				c.convertResources(nestedResources, resourcePath)
			}
		}
	}
}

// convertMethods converts methods for a specific resource path.
func (c *GoogleApiToOpenApiConverter) convertMethods(methods map[string]interface{}, resourcePath string) {
	paths := c.openApiSpec["paths"].(map[string]interface{})

	for _, methodData := range methods {
		if methodDataMap, ok := methodData.(map[string]interface{}); ok {
			httpMethod := "get"
			if m, ok := methodDataMap["httpMethod"].(string); ok {
				httpMethod = strings.ToLower(m)
			}

			// Determine the actual endpoint path
			restPath := "/"
			if path, ok := methodDataMap["path"].(string); ok {
				if !strings.HasPrefix(path, "/") {
					restPath = "/" + path
				} else {
					restPath = path
				}
			}

			pathParams := c.extractPathParameters(restPath)

			// Create path entry if it doesn't exist
			if _, ok := paths[restPath]; !ok {
				paths[restPath] = map[string]interface{}{}
			}

			// Add the operation for this method
			pathMap := paths[restPath].(map[string]interface{})
			pathMap[httpMethod] = c.convertOperation(methodDataMap, pathParams)
		}
	}
}

// extractPathParameters extracts path parameters from a URL path.
func (c *GoogleApiToOpenApiConverter) extractPathParameters(path string) []string {
	var params []string
	segments := strings.Split(path, "/")

	for _, segment := range segments {
		// Google APIs often use {param} format for path parameters
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			paramName := segment[1 : len(segment)-1]
			params = append(params, paramName)
		}
	}

	return params
}

// convertOperation converts a Google API method to an OpenAPI operation.
func (c *GoogleApiToOpenApiConverter) convertOperation(methodData map[string]interface{}, pathParams []string) map[string]interface{} {
	id := ""
	if val, ok := methodData["id"].(string); ok {
		id = val
	}

	desc := ""
	if val, ok := methodData["description"].(string); ok {
		desc = val
	}

	operation := map[string]interface{}{
		"operationId": id,
		"summary":     desc,
		"description": desc,
		"parameters":  []interface{}{},
		"responses": map[string]interface{}{
			"200": map[string]interface{}{"description": "Successful operation"},
			"400": map[string]interface{}{"description": "Bad request"},
			"401": map[string]interface{}{"description": "Unauthorized"},
			"403": map[string]interface{}{"description": "Forbidden"},
			"404": map[string]interface{}{"description": "Not found"},
			"500": map[string]interface{}{"description": "Server error"},
		},
	}

	parameters := []interface{}{}

	// Add path parameters
	for _, paramName := range pathParams {
		param := map[string]interface{}{
			"name":     paramName,
			"in":       "path",
			"required": true,
			"schema":   map[string]interface{}{"type": "string"},
		}
		parameters = append(parameters, param)
	}

	// Add query parameters
	if params, ok := methodData["parameters"].(map[string]interface{}); ok {
		for paramName, paramData := range params {
			// Skip parameters already included in path
			isPathParam := false
			for _, pathParam := range pathParams {
				if paramName == pathParam {
					isPathParam = true
					break
				}
			}
			if isPathParam {
				continue
			}

			if paramDataMap, ok := paramData.(map[string]interface{}); ok {
				desc := ""
				if val, ok := paramDataMap["description"].(string); ok {
					desc = val
				}

				required := false
				if val, ok := paramDataMap["required"].(bool); ok {
					required = val
				}

				param := map[string]interface{}{
					"name":        paramName,
					"in":          "query",
					"description": desc,
					"required":    required,
					"schema":      c.convertParameterSchema(paramDataMap),
				}
				parameters = append(parameters, param)
			}
		}
	}

	operation["parameters"] = parameters

	// Handle request body
	if request, ok := methodData["request"].(map[string]interface{}); ok {
		if ref, ok := request["$ref"].(string); ok {
			openApiRef := ""
			if strings.HasPrefix(ref, "#") {
				// Convert Google's reference format to OpenAPI format
				openApiRef = strings.Replace(ref, "#", "#/components/schemas/", 1)
			} else {
				openApiRef = "#/components/schemas/" + ref
			}

			operation["requestBody"] = map[string]interface{}{
				"description": "Request body",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"$ref": openApiRef,
						},
					},
				},
				"required": true,
			}
		}
	}

	// Handle response body
	if response, ok := methodData["response"].(map[string]interface{}); ok {
		if ref, ok := response["$ref"].(string); ok {
			openApiRef := ""
			if strings.HasPrefix(ref, "#") {
				// Convert Google's reference format to OpenAPI format
				openApiRef = strings.Replace(ref, "#", "#/components/schemas/", 1)
			} else {
				openApiRef = "#/components/schemas/" + ref
			}

			responseMap := operation["responses"].(map[string]interface{})
			successResponse := responseMap["200"].(map[string]interface{})
			successResponse["content"] = map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": openApiRef,
					},
				},
			}
		}
	}

	// Add scopes if available
	if scopes, ok := methodData["scopes"].([]interface{}); ok && len(scopes) > 0 {
		// Convert scopes to strings
		scopeStrings := make([]string, 0, len(scopes))
		for _, scope := range scopes {
			if scopeStr, ok := scope.(string); ok {
				scopeStrings = append(scopeStrings, scopeStr)
			}
		}

		// Add method-specific security requirement if different from global
		operation["security"] = []interface{}{
			map[string]interface{}{
				"oauth2": scopeStrings,
			},
		}
	}

	return operation
}

// convertParameterSchema converts a parameter definition to an OpenAPI schema.
func (c *GoogleApiToOpenApiConverter) convertParameterSchema(paramData map[string]interface{}) map[string]interface{} {
	schema := map[string]interface{}{}

	// Convert type (default to string if not specified)
	paramType := "string"
	if t, ok := paramData["type"].(string); ok && t != "" {
		paramType = t
	}
	schema["type"] = paramType

	// Handle enum values
	if enum, ok := paramData["enum"].([]interface{}); ok {
		schema["enum"] = enum
	}

	// Handle format
	if format, ok := paramData["format"].(string); ok {
		schema["format"] = format
	}

	// Handle default value
	if defaultVal, ok := paramData["default"]; ok {
		schema["default"] = defaultVal
	}

	// Handle pattern
	if pattern, ok := paramData["pattern"].(string); ok {
		schema["pattern"] = pattern
	}

	return schema
}

// SaveOpenApiSpec saves the OpenAPI specification to a file.
func (c *GoogleApiToOpenApiConverter) SaveOpenApiSpec(outputPath string) error {
	data, err := json.MarshalIndent(c.openApiSpec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	err = os.WriteFile(outputPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("OpenAPI specification saved to %s", outputPath)
	return nil
}
