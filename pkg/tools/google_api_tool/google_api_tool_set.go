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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/nvcnvn/adk-golang/pkg/auth"
	"github.com/nvcnvn/adk-golang/pkg/tools/openapi_tool/openapi_spec_parser"
)

// GoogleApiToolSet represents a collection of Google API tools.
type GoogleApiToolSet struct {
	tools []*GoogleApiTool
}

// NewGoogleApiToolSet creates a new GoogleApiToolSet from a slice of RestApiTools.
func NewGoogleApiToolSet(restApiTools []*openapi_spec_parser.RestApiTool) *GoogleApiToolSet {
	tools := make([]*GoogleApiTool, 0, len(restApiTools))
	for _, tool := range restApiTools {
		tools = append(tools, NewGoogleApiTool(tool))
	}
	return &GoogleApiToolSet{tools: tools}
}

// GetTools returns all tools in the toolset.
func (g *GoogleApiToolSet) GetTools() []*GoogleApiTool {
	return g.tools
}

// GetTool returns a tool by name, or nil if no tool with that name exists.
func (g *GoogleApiToolSet) GetTool(toolName string) *GoogleApiTool {
	for _, tool := range g.tools {
		if tool.Name() == toolName {
			return tool
		}
	}
	return nil
}

// ConfigureAuth configures OAuth credentials for all tools in the set.
func (g *GoogleApiToolSet) ConfigureAuth(clientID, clientSecret string) {
	for _, tool := range g.tools {
		tool.ConfigureAuth(clientID, clientSecret)
	}
}

// LoadToolSet loads a tool set for the specified Google API.
func LoadToolSet(apiName, apiVersion string) (*GoogleApiToolSet, error) {
	// Convert Google API to OpenAPI format
	converter := NewGoogleApiToOpenApiConverter(apiName, apiVersion)
	spec, err := converter.Convert()
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s API: %w", apiName, err)
	}

	// Extract the first scope from the spec
	var scope string
	if componentsMap, ok := spec["components"].(map[string]interface{}); ok {
		if securitySchemes, ok := componentsMap["securitySchemes"].(map[string]interface{}); ok {
			if oauth2, ok := securitySchemes["oauth2"].(map[string]interface{}); ok {
				if flows, ok := oauth2["flows"].(map[string]interface{}); ok {
					if authCode, ok := flows["authorizationCode"].(map[string]interface{}); ok {
						if scopes, ok := authCode["scopes"].(map[string]interface{}); ok {
							for s := range scopes {
								scope = s
								break
							}
						}
					}
				}
			}
		}
	}

	if scope == "" {
		log.Printf("Warning: No scope found for %s API. Authentication might fail.", apiName)
	}

	// Create OpenAPI toolset with Google OAuth config
	authScheme := &auth.OpenIDConnectWithConfig{
		Type:                  auth.OpenIDConnectScheme,
		AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenEndpoint:         "https://oauth2.googleapis.com/token",
		UserinfoEndpoint:      "https://openidconnect.googleapis.com/v1/userinfo",
		RevocationEndpoint:    "https://oauth2.googleapis.com/revoke",
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_post",
			"client_secret_basic",
		},
		GrantTypesSupported: []string{"authorization_code"},
		Scopes:              []string{scope},
	}

	// Create OpenAPI toolset using the converter output
	toolSet := openapi_spec_parser.NewOpenAPIToolset(spec, authScheme, auth.AuthCredential{
		AuthType: auth.OpenIDConnect,
		OAuth2: &auth.OAuth2Auth{
			ClientID:     "your-client-id",
			ClientSecret: "your-client-secret",
		},
	})
	toolsList := toolSet.GetTools()

	// Create Google API tool set using the OpenAPI tools
	apiTools := make([]*openapi_spec_parser.RestApiTool, 0)
	for _, tool := range toolsList {
		if restTool, ok := tool.(*openapi_spec_parser.RestApiTool); ok {
			apiTools = append(apiTools, restTool)
		}
	}

	return NewGoogleApiToolSet(apiTools), nil
}

// loadSpecFromFile loads an OpenAPI spec from a file relative to the caller.
func loadSpecFromFile(filename string) ([]byte, error) {
	// Get the caller's filename
	_, callerFile, _, ok := runtime.Caller(1)
	if !ok {
		return nil, fmt.Errorf("failed to get caller information")
	}

	// Get the directory of the caller file
	callerDir := filepath.Dir(callerFile)

	// Join with the spec filename
	specPath := filepath.Join(callerDir, filename)

	// Read and return the file contents
	return os.ReadFile(specPath)
}
