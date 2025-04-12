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

package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// IntegrationClient provides methods for interacting with Google Cloud Application Integration
type IntegrationClient struct {
	project          string
	location         string
	integration      string
	trigger          string
	connection       string
	entityOperations map[string][]string
	actions          []string
	saJSONStr        string
	tokenCache       *oauth2.Token
	tokenExpiration  time.Time
	httpClient       *http.Client
}

// NewIntegrationClient creates a new IntegrationClient instance
func NewIntegrationClient(
	project, location, integration, trigger, connection string,
	entityOperations map[string][]string, actions []string, serviceAccountJSONStr string,
) *IntegrationClient {
	// Initialize with empty maps/slices if nil
	if entityOperations == nil {
		entityOperations = make(map[string][]string)
	}
	if actions == nil {
		actions = make([]string, 0)
	}

	return &IntegrationClient{
		project:          project,
		location:         location,
		integration:      integration,
		trigger:          trigger,
		connection:       connection,
		entityOperations: entityOperations,
		actions:          actions,
		saJSONStr:        serviceAccountJSONStr,
		httpClient:       http.DefaultClient,
	}
}

// GetOpenAPISpecForIntegration generates and retrieves an OpenAPI specification for the integration
func (c *IntegrationClient) GetOpenAPISpecForIntegration() (map[string]interface{}, error) {
	url := fmt.Sprintf(
		"https://%s-integrations.googleapis.com/v1/projects/%s/locations/%s:generateOpenApiSpec",
		c.location, c.project, c.location,
	)

	// Prepare request payload
	payload := map[string]interface{}{
		"apiTriggerResources": []map[string]interface{}{
			{
				"integrationResource": c.integration,
				"triggerId":           []string{c.trigger},
			},
		},
		"fileFormat": "JSON",
	}

	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Get access token and set headers
	token, err := c.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorResp)

		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
			return nil, fmt.Errorf(
				"invalid request. Please check the provided values of project(%s), location(%s), integration(%s) and trigger(%s). Status: %d",
				c.project, c.location, c.integration, c.trigger, resp.StatusCode)
		}

		return nil, fmt.Errorf("API error (status %d): %v", resp.StatusCode, errorResp)
	}

	// Process response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract and parse the OpenAPI spec
	specJson, ok := response["openApiSpec"].(string)
	if !ok {
		return nil, fmt.Errorf("no OpenAPI spec found in response")
	}

	var spec map[string]interface{}
	if err := json.Unmarshal([]byte(specJson), &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	return spec, nil
}

// GetOpenAPISpecForConnection generates an OpenAPI specification for connector operations
func (c *IntegrationClient) GetOpenAPISpecForConnection(
	toolName, toolInstructions string,
) (map[string]interface{}, error) {
	// Application Integration needs to be provisioned in the same region as connection
	// and an integration with name "ExecuteConnection" and trigger "api_trigger/ExecuteConnection"
	// should be created as per the documentation.
	integrationName := "ExecuteConnection"

	// Create connections client
	connectionsClient := NewConnectionsClient(
		c.project,
		c.location,
		c.connection,
		c.saJSONStr,
	)

	// Validate we have entity operations or actions
	if len(c.entityOperations) == 0 && len(c.actions) == 0 {
		return nil, fmt.Errorf("no entity operations or actions provided. Please provide at least one of them")
	}

	// Get base connector spec
	connectorSpec := connectionsClient.GetConnectorBaseSpec()
	componentsSchemas := connectorSpec["components"].(map[string]interface{})["schemas"].(map[string]interface{})
	paths := connectorSpec["paths"].(map[string]interface{})

	// Process entity operations
	for entity, operations := range c.entityOperations {
		// Get entity schema and operations
		schema, supportedOperations, err := connectionsClient.GetEntitySchemaAndOperations(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to get entity schema and operations for %s: %w", entity, err)
		}

		// If no specific operations provided, use all supported operations
		if len(operations) == 0 {
			operations = supportedOperations
		}

		// Convert schema to JSON string for descriptions
		schemaBytes, err := json.Marshal(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal entity schema: %w", err)
		}
		schemaAsString := string(schemaBytes)

		// Create entity payload schema
		entityLower := entity // In Go we don't need to call toLower()
		componentsSchemas[fmt.Sprintf("connectorInputPayload_%s", entityLower)] = connectionsClient.ConnectorPayload(schema)

		// Process each operation
		for _, operation := range operations {
			operationLower := operation // Use lowercase in path
			path := fmt.Sprintf("/v2/projects/%s/locations/%s/integrations/%s:execute?triggerId=api_trigger/%s#%s_%s",
				c.project, c.location, integrationName, integrationName, operationLower, entityLower)

			switch operationLower {
			case "CREATE":
				paths[path] = connectionsClient.CreateOperation(entityLower, toolName, toolInstructions)
				componentsSchemas[fmt.Sprintf("create_%s_Request", entityLower)] = connectionsClient.CreateOperationRequest(entityLower)

			case "UPDATE":
				paths[path] = connectionsClient.UpdateOperation(entityLower, toolName, toolInstructions)
				componentsSchemas[fmt.Sprintf("update_%s_Request", entityLower)] = connectionsClient.UpdateOperationRequest(entityLower)

			case "DELETE":
				paths[path] = connectionsClient.DeleteOperation(entityLower, toolName, toolInstructions)
				componentsSchemas[fmt.Sprintf("delete_%s_Request", entityLower)] = connectionsClient.DeleteOperationRequest()

			case "LIST":
				paths[path] = connectionsClient.ListOperation(entityLower, schemaAsString, toolName, toolInstructions)
				componentsSchemas[fmt.Sprintf("list_%s_Request", entityLower)] = connectionsClient.ListOperationRequest()

			case "GET":
				paths[path] = connectionsClient.GetOperation(entityLower, schemaAsString, toolName, toolInstructions)
				componentsSchemas[fmt.Sprintf("get_%s_Request", entityLower)] = connectionsClient.GetOperationRequest()

			default:
				return nil, fmt.Errorf("invalid operation: %s for entity: %s", operation, entity)
			}
		}
	}

	// Process actions
	for _, action := range c.actions {
		// Get action schema
		actionDetails, err := connectionsClient.GetActionSchema(action)
		if err != nil {
			return nil, fmt.Errorf("failed to get action schema for %s: %w", action, err)
		}

		// Extract details
		inputSchema := actionDetails.InputSchema
		outputSchema := actionDetails.OutputSchema
		actionDisplayName := actionDetails.DisplayName

		// Set operation type based on action
		operation := "EXECUTE_ACTION"
		if action == "ExecuteCustomQuery" {
			componentsSchemas[fmt.Sprintf("%s_Request", action)] = connectionsClient.ExecuteCustomQueryRequest()
			operation = "EXECUTE_QUERY"
		} else {
			componentsSchemas[fmt.Sprintf("%s_Request", actionDisplayName)] = connectionsClient.ActionRequest(actionDisplayName)
			componentsSchemas[fmt.Sprintf("connectorInputPayload_%s", actionDisplayName)] = connectionsClient.ConnectorPayload(inputSchema)
		}

		// Add output schemas
		componentsSchemas[fmt.Sprintf("connectorOutputPayload_%s", actionDisplayName)] = connectionsClient.ConnectorPayload(outputSchema)
		componentsSchemas[fmt.Sprintf("%s_Response", actionDisplayName)] = connectionsClient.ActionResponse(actionDisplayName)

		// Create path
		path := fmt.Sprintf("/v2/projects/%s/locations/%s/integrations/%s:execute?triggerId=api_trigger/%s#%s",
			c.project, c.location, integrationName, integrationName, action)
		paths[path] = connectionsClient.GetActionOperation(action, operation, actionDisplayName, toolName, toolInstructions)
	}

	return connectorSpec, nil
}

// getAccessToken retrieves an OAuth2 access token
func (c *IntegrationClient) getAccessToken() (string, error) {
	// Return cached token if still valid
	if c.tokenCache != nil && time.Now().Before(c.tokenExpiration) {
		return c.tokenCache.AccessToken, nil
	}

	ctx := context.Background()
	var ts oauth2.TokenSource
	var err error

	if c.saJSONStr != "" {
		// Use provided service account JSON
		config, err := google.JWTConfigFromJSON(
			[]byte(c.saJSONStr),
			"https://www.googleapis.com/auth/cloud-platform",
		)
		if err != nil {
			return "", fmt.Errorf("failed to parse service account JSON: %w", err)
		}
		ts = config.TokenSource(ctx)
	} else {
		// Use default credentials
		creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return "", fmt.Errorf("failed to get default credentials: %w", err)
		}
		ts = creds.TokenSource
	}

	token, err := ts.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	// Cache the token
	c.tokenCache = token
	c.tokenExpiration = token.Expiry.Add(-5 * time.Minute) // Buffer to prevent using expired tokens

	return token.AccessToken, nil
}
