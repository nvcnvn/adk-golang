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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// ConnectionsClient is a client for interacting with Google Cloud Connectors API
type ConnectionsClient struct {
	project         string
	location        string
	connection      string
	connectorURL    string
	saJSONStr       string
	tokenCache      *oauth2.Token
	tokenExpiration time.Time
	httpClient      *http.Client
}

// NewConnectionsClient creates a new ConnectionsClient instance
func NewConnectionsClient(
	project, location, connection, serviceAccountJSONStr string,
) *ConnectionsClient {
	return &ConnectionsClient{
		project:      project,
		location:     location,
		connection:   connection,
		connectorURL: "https://connectors.googleapis.com",
		saJSONStr:    serviceAccountJSONStr,
		httpClient:   http.DefaultClient,
	}
}

// GetConnectionDetails retrieves service details for a given connection
func (c *ConnectionsClient) GetConnectionDetails() (*ConnectionDetails, error) {
	url := fmt.Sprintf(
		"%s/v1/projects/%s/locations/%s/connections/%s?view=BASIC",
		c.connectorURL, c.project, c.location, c.connection,
	)

	response, err := c.executeAPICall(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection details: %w", err)
	}
	defer response.Body.Close()

	var connData map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&connData); err != nil {
		return nil, fmt.Errorf("failed to decode connection details: %w", err)
	}

	// Extract service name and host
	serviceName, _ := connData["serviceDirectory"].(string)
	host, _ := connData["host"].(string)

	// If host is not empty, use tlsServiceDirectory as serviceName
	if host != "" {
		if tlsServiceDirectory, ok := connData["tlsServiceDirectory"].(string); ok {
			serviceName = tlsServiceDirectory
		}
	}

	authOverrideEnabled, _ := connData["authOverrideEnabled"].(bool)

	return &ConnectionDetails{
		ServiceName:         serviceName,
		Host:                host,
		AuthOverrideEnabled: authOverrideEnabled,
	}, nil
}

// GetEntitySchemaAndOperations retrieves the JSON schema and supported operations for a given entity
func (c *ConnectionsClient) GetEntitySchemaAndOperations(
	entity string,
) (map[string]interface{}, []string, error) {
	url := fmt.Sprintf(
		"%s/v1/projects/%s/locations/%s/connections/%s/connectionSchemaMetadata:getEntityType?entityId=%s",
		c.connectorURL, c.project, c.location, c.connection, entity,
	)

	response, err := c.executeAPICall(url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get entity schema: %w", err)
	}
	defer response.Body.Close()

	var respData map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&respData); err != nil {
		return nil, nil, fmt.Errorf("failed to decode entity schema response: %w", err)
	}

	operationID, ok := respData["name"].(string)
	if !ok || operationID == "" {
		return nil, nil, fmt.Errorf("failed to get operation ID for entity: %s", entity)
	}

	// Poll the operation until complete
	operation, err := c.pollOperation(operationID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to poll operation: %w", err)
	}

	responseData, ok := operation.Response["response"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("invalid operation response format")
	}

	// Extract schema and operations
	schema, _ := responseData["jsonSchema"].(map[string]interface{})

	var operations []string
	if opsRaw, ok := responseData["operations"].([]interface{}); ok {
		operations = make([]string, 0, len(opsRaw))
		for _, op := range opsRaw {
			if opStr, ok := op.(string); ok {
				operations = append(operations, opStr)
			}
		}
	}

	return schema, operations, nil
}

// GetActionSchema retrieves the input and output JSON schema for a given action
func (c *ConnectionsClient) GetActionSchema(action string) (*ActionDetails, error) {
	url := fmt.Sprintf(
		"%s/v1/projects/%s/locations/%s/connections/%s/connectionSchemaMetadata:getAction?actionId=%s",
		c.connectorURL, c.project, c.location, c.connection, action,
	)

	response, err := c.executeAPICall(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get action schema: %w", err)
	}
	defer response.Body.Close()

	var respData map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&respData); err != nil {
		return nil, fmt.Errorf("failed to decode action schema response: %w", err)
	}

	operationID, ok := respData["name"].(string)
	if !ok || operationID == "" {
		return nil, fmt.Errorf("failed to get operation ID for action: %s", action)
	}

	// Poll the operation until complete
	operation, err := c.pollOperation(operationID)
	if err != nil {
		return nil, fmt.Errorf("failed to poll operation: %w", err)
	}

	responseData, ok := operation.Response["response"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid operation response format")
	}

	// Extract the schemas and details
	inputSchema, _ := responseData["inputJsonSchema"].(map[string]interface{})
	outputSchema, _ := responseData["outputJsonSchema"].(map[string]interface{})
	description, _ := responseData["description"].(string)
	displayName, _ := responseData["displayName"].(string)

	return &ActionDetails{
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
		Description:  description,
		DisplayName:  displayName,
	}, nil
}

// GetConnectorBaseSpec returns the base OpenAPI specification for the connector
func (c *ConnectionsClient) GetConnectorBaseSpec() map[string]interface{} {
	return map[string]interface{}{
		"openapi": "3.0.1",
		"info": map[string]interface{}{
			"title":       "ExecuteConnection",
			"description": "This tool can execute a query on connection",
			"version":     "4",
		},
		"servers": []map[string]interface{}{
			{"url": "https://integrations.googleapis.com"},
		},
		"security": []map[string]interface{}{
			{"google_auth": []string{"https://www.googleapis.com/auth/cloud-platform"}},
		},
		"paths": map[string]interface{}{},
		"components": map[string]interface{}{
			"schemas": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"default":     "LIST_ENTITIES",
					"description": "Operation to execute. Possible values are LIST_ENTITIES, GET_ENTITY, CREATE_ENTITY, UPDATE_ENTITY, DELETE_ENTITY in case of entities. EXECUTE_ACTION in case of actions. and EXECUTE_QUERY in case of custom queries.",
				},
				"entityId": map[string]interface{}{
					"type":        "string",
					"description": "Name of the entity",
				},
				"connectorInputPayload": map[string]interface{}{
					"type": "object",
				},
				"filterClause": map[string]interface{}{
					"type":        "string",
					"default":     "",
					"description": "WHERE clause in SQL query",
				},
				"pageSize": map[string]interface{}{
					"type":        "integer",
					"default":     50,
					"description": "Number of entities to return in the response",
				},
				"pageToken": map[string]interface{}{
					"type":        "string",
					"default":     "",
					"description": "Page token to return the next page of entities",
				},
				"connectionName": map[string]interface{}{
					"type":        "string",
					"default":     "",
					"description": "Connection resource name to run the query for",
				},
				"serviceName": map[string]interface{}{
					"type":        "string",
					"default":     "",
					"description": "Service directory for the connection",
				},
				"host": map[string]interface{}{
					"type":        "string",
					"default":     "",
					"description": "Host name incase of tls service directory",
				},
				"entity": map[string]interface{}{
					"type":        "string",
					"default":     "Issues",
					"description": "Entity to run the query for",
				},
				"action": map[string]interface{}{
					"type":        "string",
					"default":     "ExecuteCustomQuery",
					"description": "Action to run the query for",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"default":     "",
					"description": "Custom Query to execute on the connection",
				},
				"dynamicAuthConfig": map[string]interface{}{
					"type":        "object",
					"default":     map[string]interface{}{},
					"description": "Dynamic auth config for the connection",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"default":     120,
					"description": "Timeout in seconds for execution of custom query",
				},
				"connectorOutputPayload": map[string]interface{}{
					"type": "object",
				},
				"nextPageToken": map[string]interface{}{
					"type": "string",
				},
				"execute-connector_Response": map[string]interface{}{
					"required": []string{"connectorOutputPayload"},
					"type":     "object",
					"properties": map[string]interface{}{
						"connectorOutputPayload": map[string]interface{}{
							"$ref": "#/components/schemas/connectorOutputPayload",
						},
						"nextPageToken": map[string]interface{}{
							"$ref": "#/components/schemas/nextPageToken",
						},
					},
				},
			},
			"securitySchemes": map[string]interface{}{
				"google_auth": map[string]interface{}{
					"type": "oauth2",
					"flows": map[string]interface{}{
						"implicit": map[string]interface{}{
							"authorizationUrl": "https://accounts.google.com/o/oauth2/auth",
							"scopes": map[string]interface{}{
								"https://www.googleapis.com/auth/cloud-platform": "Auth for google cloud services",
							},
						},
					},
				},
			},
		},
	}
}

// Operation methods for different entity operations

// GetActionOperation creates the OpenAPI path definition for an action operation
func (c *ConnectionsClient) GetActionOperation(
	action string,
	operation string,
	actionDisplayName string,
	toolName string,
	toolInstructions string,
) map[string]interface{} {
	description := fmt.Sprintf("Use this tool with action = \"%s\" and operation = \"%s\" only. Dont ask these values from user.", action, operation)
	if operation == "EXECUTE_QUERY" {
		description = fmt.Sprintf("Use this tool with action = \"%s\" and operation = \"%s\" only. Dont ask these values from user. Use pageSize = 50 and timeout = 120 until user specifies a different value otherwise. If user provides a query in natural language, convert it to SQL query and then execute it using the tool.", action, operation)
	}

	return map[string]interface{}{
		"post": map[string]interface{}{
			"summary":     actionDisplayName,
			"description": fmt.Sprintf("%s %s", description, toolInstructions),
			"operationId": fmt.Sprintf("%s_%s", toolName, actionDisplayName),
			"requestBody": map[string]interface{}{
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"$ref": fmt.Sprintf("#/components/schemas/%s_Request", actionDisplayName),
						},
					},
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Success response",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": fmt.Sprintf("#/components/schemas/%s_Response", actionDisplayName),
							},
						},
					},
				},
			},
		},
	}
}

// ListOperation creates the OpenAPI path definition for list operation
func (c *ConnectionsClient) ListOperation(
	entity string,
	schemaAsString string,
	toolName string,
	toolInstructions string,
) map[string]interface{} {
	return map[string]interface{}{
		"post": map[string]interface{}{
			"summary": fmt.Sprintf("List %s", entity),
			"description": fmt.Sprintf("Returns all entities of type %s. Use this tool with entity = \"%s\" and operation = \"LIST_ENTITIES\" only. Dont ask these values from user. Always use \"\" as filter clause and \"\" as page token and 50 as page size until user specifies a different value otherwise. Use single quotes for strings in filter clause. %s",
				entity, entity, toolInstructions),
			"operationId": fmt.Sprintf("%s_list_%s", toolName, entity),
			"requestBody": map[string]interface{}{
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"$ref": fmt.Sprintf("#/components/schemas/list_%s_Request", entity),
						},
					},
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Success response",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"description": fmt.Sprintf("Returns a list of %s of json schema: %s", entity, schemaAsString),
								"$ref":        "#/components/schemas/execute-connector_Response",
							},
						},
					},
				},
			},
		},
	}
}

// GetOperation creates the OpenAPI path definition for get operation
func (c *ConnectionsClient) GetOperation(
	entity string,
	schemaAsString string,
	toolName string,
	toolInstructions string,
) map[string]interface{} {
	return map[string]interface{}{
		"post": map[string]interface{}{
			"summary": fmt.Sprintf("Get %s", entity),
			"description": fmt.Sprintf("Returns the details of the %s. Use this tool with entity = \"%s\" and operation = \"GET_ENTITY\" only. Dont ask these values from user. %s",
				entity, entity, toolInstructions),
			"operationId": fmt.Sprintf("%s_get_%s", toolName, entity),
			"requestBody": map[string]interface{}{
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"$ref": fmt.Sprintf("#/components/schemas/get_%s_Request", entity),
						},
					},
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Success response",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"description": fmt.Sprintf("Returns %s of json schema: %s", entity, schemaAsString),
								"$ref":        "#/components/schemas/execute-connector_Response",
							},
						},
					},
				},
			},
		},
	}
}

// CreateOperation creates the OpenAPI path definition for create operation
func (c *ConnectionsClient) CreateOperation(
	entity string,
	toolName string,
	toolInstructions string,
) map[string]interface{} {
	return map[string]interface{}{
		"post": map[string]interface{}{
			"summary": fmt.Sprintf("Create %s", entity),
			"description": fmt.Sprintf("Creates a new entity of type %s. Use this tool with entity = \"%s\" and operation = \"CREATE_ENTITY\" only. Dont ask these values from user. Follow the schema of the entity provided in the instructions to create %s. %s",
				entity, entity, entity, toolInstructions),
			"operationId": fmt.Sprintf("%s_create_%s", toolName, entity),
			"requestBody": map[string]interface{}{
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"$ref": fmt.Sprintf("#/components/schemas/create_%s_Request", entity),
						},
					},
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Success response",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/execute-connector_Response",
							},
						},
					},
				},
			},
		},
	}
}

// UpdateOperation creates the OpenAPI path definition for update operation
func (c *ConnectionsClient) UpdateOperation(
	entity string,
	toolName string,
	toolInstructions string,
) map[string]interface{} {
	return map[string]interface{}{
		"post": map[string]interface{}{
			"summary": fmt.Sprintf("Update %s", entity),
			"description": fmt.Sprintf("Updates an entity of type %s. Use this tool with entity = \"%s\" and operation = \"UPDATE_ENTITY\" only. Dont ask these values from user. Use entityId to uniquely identify the entity to update. Follow the schema of the entity provided in the instructions to update %s. %s",
				entity, entity, entity, toolInstructions),
			"operationId": fmt.Sprintf("%s_update_%s", toolName, entity),
			"requestBody": map[string]interface{}{
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"$ref": fmt.Sprintf("#/components/schemas/update_%s_Request", entity),
						},
					},
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Success response",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/execute-connector_Response",
							},
						},
					},
				},
			},
		},
	}
}

// DeleteOperation creates the OpenAPI path definition for delete operation
func (c *ConnectionsClient) DeleteOperation(
	entity string,
	toolName string,
	toolInstructions string,
) map[string]interface{} {
	return map[string]interface{}{
		"post": map[string]interface{}{
			"summary": fmt.Sprintf("Delete %s", entity),
			"description": fmt.Sprintf("Deletes an entity of type %s. Use this tool with entity = \"%s\" and operation = \"DELETE_ENTITY\" only. Dont ask these values from user. %s",
				entity, entity, toolInstructions),
			"operationId": fmt.Sprintf("%s_delete_%s", toolName, entity),
			"requestBody": map[string]interface{}{
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"$ref": fmt.Sprintf("#/components/schemas/delete_%s_Request", entity),
						},
					},
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Success response",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"$ref": "#/components/schemas/execute-connector_Response",
							},
						},
					},
				},
			},
		},
	}
}

// Request schemas for different operations

// CreateOperationRequest creates the request schema for create operation
func (c *ConnectionsClient) CreateOperationRequest(entity string) map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"required": []string{
			"connectorInputPayload",
			"operation",
			"connectionName",
			"serviceName",
			"host",
			"entity",
		},
		"properties": map[string]interface{}{
			"connectorInputPayload": map[string]interface{}{
				"$ref": fmt.Sprintf("#/components/schemas/connectorInputPayload_%s", entity),
			},
			"operation":      map[string]interface{}{"$ref": "#/components/schemas/operation"},
			"connectionName": map[string]interface{}{"$ref": "#/components/schemas/connectionName"},
			"serviceName":    map[string]interface{}{"$ref": "#/components/schemas/serviceName"},
			"host":           map[string]interface{}{"$ref": "#/components/schemas/host"},
			"entity":         map[string]interface{}{"$ref": "#/components/schemas/entity"},
		},
	}
}

// UpdateOperationRequest creates the request schema for update operation
func (c *ConnectionsClient) UpdateOperationRequest(entity string) map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"required": []string{
			"connectorInputPayload",
			"entityId",
			"operation",
			"connectionName",
			"serviceName",
			"host",
			"entity",
		},
		"properties": map[string]interface{}{
			"connectorInputPayload": map[string]interface{}{
				"$ref": fmt.Sprintf("#/components/schemas/connectorInputPayload_%s", entity),
			},
			"entityId":       map[string]interface{}{"$ref": "#/components/schemas/entityId"},
			"operation":      map[string]interface{}{"$ref": "#/components/schemas/operation"},
			"connectionName": map[string]interface{}{"$ref": "#/components/schemas/connectionName"},
			"serviceName":    map[string]interface{}{"$ref": "#/components/schemas/serviceName"},
			"host":           map[string]interface{}{"$ref": "#/components/schemas/host"},
			"entity":         map[string]interface{}{"$ref": "#/components/schemas/entity"},
		},
	}
}

// GetOperationRequest creates the request schema for get operation
func (c *ConnectionsClient) GetOperationRequest() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"required": []string{
			"entityId",
			"operation",
			"connectionName",
			"serviceName",
			"host",
			"entity",
		},
		"properties": map[string]interface{}{
			"entityId":       map[string]interface{}{"$ref": "#/components/schemas/entityId"},
			"operation":      map[string]interface{}{"$ref": "#/components/schemas/operation"},
			"connectionName": map[string]interface{}{"$ref": "#/components/schemas/connectionName"},
			"serviceName":    map[string]interface{}{"$ref": "#/components/schemas/serviceName"},
			"host":           map[string]interface{}{"$ref": "#/components/schemas/host"},
			"entity":         map[string]interface{}{"$ref": "#/components/schemas/entity"},
		},
	}
}

// DeleteOperationRequest creates the request schema for delete operation
func (c *ConnectionsClient) DeleteOperationRequest() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"required": []string{
			"entityId",
			"operation",
			"connectionName",
			"serviceName",
			"host",
			"entity",
		},
		"properties": map[string]interface{}{
			"entityId":       map[string]interface{}{"$ref": "#/components/schemas/entityId"},
			"operation":      map[string]interface{}{"$ref": "#/components/schemas/operation"},
			"connectionName": map[string]interface{}{"$ref": "#/components/schemas/connectionName"},
			"serviceName":    map[string]interface{}{"$ref": "#/components/schemas/serviceName"},
			"host":           map[string]interface{}{"$ref": "#/components/schemas/host"},
			"entity":         map[string]interface{}{"$ref": "#/components/schemas/entity"},
		},
	}
}

// ListOperationRequest creates the request schema for list operation
func (c *ConnectionsClient) ListOperationRequest() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"required": []string{
			"operation",
			"connectionName",
			"serviceName",
			"host",
			"entity",
		},
		"properties": map[string]interface{}{
			"filterClause":   map[string]interface{}{"$ref": "#/components/schemas/filterClause"},
			"pageSize":       map[string]interface{}{"$ref": "#/components/schemas/pageSize"},
			"pageToken":      map[string]interface{}{"$ref": "#/components/schemas/pageToken"},
			"operation":      map[string]interface{}{"$ref": "#/components/schemas/operation"},
			"connectionName": map[string]interface{}{"$ref": "#/components/schemas/connectionName"},
			"serviceName":    map[string]interface{}{"$ref": "#/components/schemas/serviceName"},
			"host":           map[string]interface{}{"$ref": "#/components/schemas/host"},
			"entity":         map[string]interface{}{"$ref": "#/components/schemas/entity"},
		},
	}
}

// ActionRequest creates the request schema for an action
func (c *ConnectionsClient) ActionRequest(action string) map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"required": []string{
			"operation",
			"connectionName",
			"serviceName",
			"host",
			"action",
			"connectorInputPayload",
		},
		"properties": map[string]interface{}{
			"operation":      map[string]interface{}{"$ref": "#/components/schemas/operation"},
			"connectionName": map[string]interface{}{"$ref": "#/components/schemas/connectionName"},
			"serviceName":    map[string]interface{}{"$ref": "#/components/schemas/serviceName"},
			"host":           map[string]interface{}{"$ref": "#/components/schemas/host"},
			"action":         map[string]interface{}{"$ref": "#/components/schemas/action"},
			"connectorInputPayload": map[string]interface{}{
				"$ref": fmt.Sprintf("#/components/schemas/connectorInputPayload_%s", action),
			},
		},
	}
}

// ActionResponse creates the response schema for an action
func (c *ConnectionsClient) ActionResponse(action string) map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"connectorOutputPayload": map[string]interface{}{
				"$ref": fmt.Sprintf("#/components/schemas/connectorOutputPayload_%s", action),
			},
		},
	}
}

// ExecuteCustomQueryRequest creates the request schema for execute custom query
func (c *ConnectionsClient) ExecuteCustomQueryRequest() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"required": []string{
			"operation",
			"connectionName",
			"serviceName",
			"host",
			"action",
			"query",
			"timeout",
			"pageSize",
		},
		"properties": map[string]interface{}{
			"operation":      map[string]interface{}{"$ref": "#/components/schemas/operation"},
			"connectionName": map[string]interface{}{"$ref": "#/components/schemas/connectionName"},
			"serviceName":    map[string]interface{}{"$ref": "#/components/schemas/serviceName"},
			"host":           map[string]interface{}{"$ref": "#/components/schemas/host"},
			"action":         map[string]interface{}{"$ref": "#/components/schemas/action"},
			"query":          map[string]interface{}{"$ref": "#/components/schemas/query"},
			"timeout":        map[string]interface{}{"$ref": "#/components/schemas/timeout"},
			"pageSize":       map[string]interface{}{"$ref": "#/components/schemas/pageSize"},
		},
	}
}

// ConnectorPayload creates a connector payload schema from a JSON schema
func (c *ConnectionsClient) ConnectorPayload(jsonSchema map[string]interface{}) map[string]interface{} {
	return c.convertJSONSchemaToOpenAPISchema(jsonSchema)
}

// convertJSONSchemaToOpenAPISchema converts a JSON schema to an OpenAPI schema
func (c *ConnectionsClient) convertJSONSchemaToOpenAPISchema(jsonSchema map[string]interface{}) map[string]interface{} {
	openapiSchema := make(map[string]interface{})

	// Copy description if present
	if desc, ok := jsonSchema["description"].(string); ok {
		openapiSchema["description"] = desc
	}

	// Handle type
	if jsonSchemaType, ok := jsonSchema["type"]; ok {
		if typeList, ok := jsonSchemaType.([]interface{}); ok {
			// Check for nullable type
			isNullable := false
			var otherTypes []string

			for _, t := range typeList {
				if tStr, ok := t.(string); ok {
					if tStr == "null" {
						isNullable = true
					} else {
						otherTypes = append(otherTypes, tStr)
					}
				}
			}

			if isNullable {
				openapiSchema["nullable"] = true
			}

			if len(otherTypes) > 0 {
				openapiSchema["type"] = otherTypes[0]
			}
		} else if typeStr, ok := jsonSchemaType.(string); ok {
			openapiSchema["type"] = typeStr
		}
	}

	// Handle object properties
	if openapiSchema["type"] == "object" {
		if props, ok := jsonSchema["properties"].(map[string]interface{}); ok {
			openapiProps := make(map[string]interface{})
			for propName, propSchema := range props {
				if propSchemaMap, ok := propSchema.(map[string]interface{}); ok {
					openapiProps[propName] = c.convertJSONSchemaToOpenAPISchema(propSchemaMap)
				}
			}
			openapiSchema["properties"] = openapiProps
		}
	}

	// Handle array items
	if openapiSchema["type"] == "array" {
		if items, ok := jsonSchema["items"]; ok {
			if itemsList, ok := items.([]interface{}); ok {
				openapiItems := make([]map[string]interface{}, 0)
				for _, item := range itemsList {
					if itemMap, ok := item.(map[string]interface{}); ok {
						openapiItems = append(openapiItems, c.convertJSONSchemaToOpenAPISchema(itemMap))
					}
				}
				openapiSchema["items"] = openapiItems
			} else if itemsMap, ok := items.(map[string]interface{}); ok {
				openapiSchema["items"] = c.convertJSONSchemaToOpenAPISchema(itemsMap)
			}
		}
	}

	return openapiSchema
}

// Helper methods for API calls

// getAccessToken gets an OAuth2 access token for API authentication
func (c *ConnectionsClient) getAccessToken() (string, error) {
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

// executeAPICall executes an API call with proper authentication
func (c *ConnectionsClient) executeAPICall(url string) (*http.Response, error) {
	token, err := c.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var errorResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorResp)

		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
			return nil, fmt.Errorf("invalid request. Please check the provided values of project(%s), location(%s), connection(%s). Status: %d",
				c.project, c.location, c.connection, resp.StatusCode)
		}

		return nil, fmt.Errorf("API error (status %d): %v", resp.StatusCode, errorResp)
	}

	return resp, nil
}

// pollOperation polls a long-running operation until it's complete
func (c *ConnectionsClient) pollOperation(operationID string) (*LongRunningOperation, error) {
	url := fmt.Sprintf("%s/v1/%s", c.connectorURL, operationID)

	for {
		response, err := c.executeAPICall(url)
		if err != nil {
			return nil, fmt.Errorf("failed to poll operation: %w", err)
		}
		defer response.Body.Close()

		var opData map[string]interface{}
		if err := json.NewDecoder(response.Body).Decode(&opData); err != nil {
			return nil, fmt.Errorf("failed to decode operation response: %w", err)
		}

		done, _ := opData["done"].(bool)
		if done {
			return &LongRunningOperation{
				Name:     operationID,
				Done:     true,
				Response: opData,
			}, nil
		}

		// Sleep before polling again
		time.Sleep(1 * time.Second)
	}
}
