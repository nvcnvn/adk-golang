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

// Package application_integration_tool contains tools for interacting with Google Cloud Application Integration
package application_integration_tool

import (
	"encoding/json"
	"fmt"

	"github.com/nvcnvn/adk-golang/pkg/auth"
	"github.com/nvcnvn/adk-golang/pkg/tools/application_integration_tool/clients"
	"github.com/nvcnvn/adk-golang/pkg/tools/openapi_tool/openapi_spec_parser"
)

// ApplicationIntegrationToolset generates tools from a given Application Integration or Integration Connector resource.
//
// Example Usage:
//
//	// Get all available tools for an integration with API trigger
//	toolset, err := NewApplicationIntegrationToolset(&ApplicationIntegrationToolsetOptions{
//		Project:               "test-project",
//		Location:              "us-central1",
//		Integration:           "test-integration",
//		Trigger:               "api_trigger/test_trigger",
//		ServiceAccountJSONStr: "...", // Optional service account JSON
//	})
//
//	// Get all available tools for a connection using entity operations and actions
//	toolset, err := NewApplicationIntegrationToolset(&ApplicationIntegrationToolsetOptions{
//		Project:          "test-project",
//		Location:         "us-central1",
//		Connection:       "test-connection",
//		EntityOperations: map[string][]string{"EntityId1": {"LIST", "CREATE"}, "EntityId2": {}},
//		Actions:          []string{"action1"},
//	})
//
//	// Get all available tools
//	tools := toolset.GetTools()
type ApplicationIntegrationToolset struct {
	project          string
	location         string
	integration      string
	trigger          string
	connection       string
	entityOperations map[string][]string
	actions          []string
	toolName         string
	toolInstructions string
	saJSONStr        string
	generatedTools   []*openapi_spec_parser.RestApiTool
}

// ApplicationIntegrationToolsetOptions contains options for creating a new ApplicationIntegrationToolset
type ApplicationIntegrationToolsetOptions struct {
	// Project ID
	Project string
	// GCP location (e.g., us-central1)
	Location string
	// Integration name (optional if Connection is provided)
	Integration string
	// Trigger name (required if Integration is provided)
	Trigger string
	// Connection name (optional if Integration is provided)
	Connection string
	// Entity operations supported by the connection (optional)
	EntityOperations map[string][]string
	// Actions supported by the connection (optional)
	Actions []string
	// Tool name prefix (optional)
	ToolName string
	// Additional tool instructions (optional)
	ToolInstructions string
	// Service account JSON string (optional)
	ServiceAccountJSONStr string
}

// NewApplicationIntegrationToolset creates a new ApplicationIntegrationToolset
func NewApplicationIntegrationToolset(opts *ApplicationIntegrationToolsetOptions) (*ApplicationIntegrationToolset, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	if opts.Project == "" || opts.Location == "" {
		return nil, fmt.Errorf("project and location are required")
	}

	// Validate either (integration and trigger) or (connection and (entityOperations or actions)) is provided
	if (opts.Integration != "" && opts.Trigger == "") ||
		(opts.Integration == "" && opts.Trigger != "") {
		return nil, fmt.Errorf("both integration and trigger must be provided together")
	}

	if opts.Integration == "" && opts.Connection == "" {
		return nil, fmt.Errorf("either integration or connection must be provided")
	}

	if opts.Connection != "" && opts.EntityOperations == nil && len(opts.Actions) == 0 {
		return nil, fmt.Errorf("either entityOperations or actions must be provided with connection")
	}

	// Initialize the toolset
	toolset := &ApplicationIntegrationToolset{
		project:          opts.Project,
		location:         opts.Location,
		integration:      opts.Integration,
		trigger:          opts.Trigger,
		connection:       opts.Connection,
		entityOperations: opts.EntityOperations,
		actions:          opts.Actions,
		toolName:         opts.ToolName,
		toolInstructions: opts.ToolInstructions,
		saJSONStr:        opts.ServiceAccountJSONStr,
	}

	// Initialize the tools
	if err := toolset.initializeTools(); err != nil {
		return nil, fmt.Errorf("failed to initialize tools: %w", err)
	}

	return toolset, nil
}

// initializeTools initializes the tools for the toolset
func (t *ApplicationIntegrationToolset) initializeTools() error {
	// Create the integration client
	integrationClient := clients.NewIntegrationClient(
		t.project,
		t.location,
		t.integration,
		t.trigger,
		t.connection,
		t.entityOperations,
		t.actions,
		t.saJSONStr,
	)

	var specDict map[string]interface{}
	var err error

	if t.integration != "" && t.trigger != "" {
		// Get OpenAPI spec for integration
		specDict, err = integrationClient.GetOpenAPISpecForIntegration()
		if err != nil {
			return fmt.Errorf("failed to get OpenAPI spec for integration: %w", err)
		}
	} else if t.connection != "" && (len(t.entityOperations) > 0 || len(t.actions) > 0) {
		// Get connection details
		connectionsClient := clients.NewConnectionsClient(
			t.project,
			t.location,
			t.connection,
			t.saJSONStr,
		)

		connDetails, err := connectionsClient.GetConnectionDetails()
		if err != nil {
			return fmt.Errorf("failed to get connection details: %w", err)
		}

		// Append connection details to tool instructions
		t.toolInstructions += fmt.Sprintf(" ALWAYS use serviceName = %s, host = %s and the connection name = %s when using this tool. DONOT ask the user for these values as you already have those.",
			connDetails.ServiceName,
			connDetails.Host,
			fmt.Sprintf("projects/%s/locations/%s/connections/%s", t.project, t.location, t.connection),
		)

		// Get OpenAPI spec for connection
		specDict, err = integrationClient.GetOpenAPISpecForConnection(t.toolName, t.toolInstructions)
		if err != nil {
			return fmt.Errorf("failed to get OpenAPI spec for connection: %w", err)
		}
	} else {
		return fmt.Errorf("invalid toolset configuration")
	}

	// Create auth credentials
	var authCred auth.AuthCredential
	var authScheme auth.AuthScheme

	// Create Bearer auth scheme
	authScheme = &auth.SecurityScheme{
		Type:         auth.HTTPScheme,
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}

	if t.saJSONStr != "" {
		// Use provided service account JSON
		var svcAcctCred auth.ServiceAccountCredential
		if err := json.Unmarshal([]byte(t.saJSONStr), &svcAcctCred); err != nil {
			return fmt.Errorf("failed to parse service account JSON: %w", err)
		}

		authCred = auth.AuthCredential{
			AuthType: auth.ServiceAccountType,
			ServiceAcct: &auth.ServiceAccountAuth{
				ServiceAccountCredential: &svcAcctCred,
				Scopes:                   []string{"https://www.googleapis.com/auth/cloud-platform"},
			},
		}
	} else {
		// Use default credentials
		authCred = auth.AuthCredential{
			AuthType: auth.ServiceAccountType,
			ServiceAcct: &auth.ServiceAccountAuth{
				UseDefaultCredential: true,
				Scopes:               []string{"https://www.googleapis.com/auth/cloud-platform"},
			},
		}
	}

	// Parse the spec to generate tools
	toolset := openapi_spec_parser.NewOpenAPIToolset(specDict, authScheme, authCred)

	// Get the tools from the toolset and convert them to the correct type
	tools := toolset.GetTools()

	t.generatedTools = make([]*openapi_spec_parser.RestApiTool, 0, len(tools))
	for _, tool := range tools {
		if restTool, ok := tool.(*openapi_spec_parser.RestApiTool); ok {
			t.generatedTools = append(t.generatedTools, restTool)
		}
	}

	return nil
}

// GetTools returns all the tools generated by this toolset
func (t *ApplicationIntegrationToolset) GetTools() []*openapi_spec_parser.RestApiTool {
	return t.generatedTools
}
