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

package auth

import (
	"context"
	"fmt"
	"sync"
)

// REQUEST_EUC_FUNCTION_CALL_NAME is the name of the function call for requesting end-user credentials
const REQUEST_EUC_FUNCTION_CALL_NAME = "request_euc"

// InvocationContext represents the context for agent invocation
type InvocationContext interface {
	// Agent returns the agent associated with this context
	Agent() interface{}

	// Session returns the session associated with this context
	Session() SessionContext
}

// SessionContext extends Session with events
type SessionContext interface {
	Session

	// Events returns the events in this session
	Events() []Event
}

// Event represents an event in a conversation
type Event interface {
	// Author returns the author of this event
	Author() string

	// GetFunctionCalls returns the function calls in this event
	GetFunctionCalls() []FunctionCall

	// GetFunctionResponses returns the function responses in this event
	GetFunctionResponses() []FunctionResponse
}

// FunctionCall represents a call to a function
type FunctionCall interface {
	// ID returns the ID of this function call
	ID() string

	// Name returns the name of the function
	Name() string

	// Args returns the arguments to the function
	Args() map[string]interface{}
}

// FunctionResponse represents a response to a function call
type FunctionResponse interface {
	// ID returns the ID of this function response
	ID() string

	// Name returns the name of the function
	Name() string

	// Response returns the response data
	Response() interface{}
}

// LLMRequest represents a request to an LLM
type LLMRequest interface {
	// ID returns the ID of this request
	ID() string
}

// FunctionCallsHandler is a function that handles function calls
type FunctionCallsHandler func(ctx context.Context, invocation InvocationContext, event Event, tools map[string]interface{}, toolsToResume map[string]struct{}) (Event, error)

// AuthLLMRequestProcessor processes LLM requests with authentication
type AuthLLMRequestProcessor struct {
	HandleFunctionCalls FunctionCallsHandler
	mu                  sync.Mutex // For thread safety
}

// NewAuthLLMRequestProcessor creates a new AuthLLMRequestProcessor
func NewAuthLLMRequestProcessor(handleFunctionCalls FunctionCallsHandler) *AuthLLMRequestProcessor {
	return &AuthLLMRequestProcessor{
		HandleFunctionCalls: handleFunctionCalls,
	}
}

// ProcessAsync processes an LLM request asynchronously
func (p *AuthLLMRequestProcessor) ProcessAsync(ctx context.Context, invocation InvocationContext, llmRequest LLMRequest) ([]Event, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	session := invocation.Session()
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	events := session.Events()
	if len(events) == 0 {
		return nil, nil
	}

	// Find request EUC function call IDs
	requestEUCFunctionCallIDs := make(map[string]struct{})

	// Look from the most recent event for a user event with function responses
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.Author() != "user" {
			continue
		}

		responses := event.GetFunctionResponses()
		if len(responses) == 0 {
			return nil, nil
		}

		for _, response := range responses {
			if response.Name() != REQUEST_EUC_FUNCTION_CALL_NAME {
				continue
			}

			// Found the function call response for the system long-running request EUC function call
			requestEUCFunctionCallIDs[response.ID()] = struct{}{}

			// Parse auth config from response
			authConfigData, ok := response.Response().(map[string]interface{})
			if !ok {
				continue
			}

			// Initialize auth config
			var authConfig AuthConfig
			if err := convertToAuthConfig(authConfigData, &authConfig); err != nil {
				return nil, err
			}

			// Initialize auth handler and store auth response
			handler := NewAuthHandler(authConfig)
			if err := handler.ParseAndStoreAuthResponse(session); err != nil {
				return nil, err
			}
		}

		// Found a user event with responses, no need to look further
		break
	}

	if len(requestEUCFunctionCallIDs) == 0 {
		return nil, nil
	}

	// Look for the system long-running request EUC function call and the original function calls
	return p.processAuthRequests(ctx, invocation, events, requestEUCFunctionCallIDs)
}

// processAuthRequests processes auth requests by finding and handling the relevant function calls
func (p *AuthLLMRequestProcessor) processAuthRequests(ctx context.Context, invocation InvocationContext, events []Event, requestEUCFunctionCallIDs map[string]struct{}) ([]Event, error) {
	// Look for the system long-running request EUC function call
	for i := len(events) - 2; i >= 0; i-- {
		event := events[i]
		functionCalls := event.GetFunctionCalls()
		if len(functionCalls) == 0 {
			continue
		}

		toolsToResume := make(map[string]struct{})

		for _, functionCall := range functionCalls {
			if _, ok := requestEUCFunctionCallIDs[functionCall.ID()]; !ok {
				continue
			}

			// Parse auth tool arguments
			argsMap := functionCall.Args()
			functionCallID, ok := argsMap["function_call_id"].(string)
			if !ok {
				continue
			}

			toolsToResume[functionCallID] = struct{}{}
		}

		if len(toolsToResume) == 0 {
			continue
		}

		// Found the system long-running request EUC function call
		// Look for original function call that requested EUC
		for j := i - 1; j >= 0; j-- {
			event := events[j]
			functionCalls := event.GetFunctionCalls()
			if len(functionCalls) == 0 {
				continue
			}

			// Get canonical tools from agent
			agent := invocation.Agent()
			canonicalTools := extractCanonicalTools(agent)

			// Check each function call
			for _, functionCall := range functionCalls {
				if _, ok := toolsToResume[functionCall.ID()]; ok {
					if p.HandleFunctionCalls != nil {
						responseEvent, err := p.HandleFunctionCalls(
							ctx,
							invocation,
							event,
							canonicalTools,
							toolsToResume,
						)
						if err != nil {
							return nil, err
						}

						if responseEvent != nil {
							return []Event{responseEvent}, nil
						}
					}
				}
			}

			// If we got here, we found the original event but couldn't process it
			return nil, nil
		}

		// If we got here, we couldn't find the original event
		return nil, nil
	}

	return nil, nil
}

// Helper function to extract canonical tools from an agent
func extractCanonicalTools(agent interface{}) map[string]interface{} {
	// In a real implementation, we would extract tools from the agent
	// Here we just return an empty map for now
	return make(map[string]interface{})
}

// Helper function to convert a map to an AuthConfig
func convertToAuthConfig(data map[string]interface{}, config *AuthConfig) error {
	// In a real implementation, we would properly deserialize the data to an AuthConfig
	// For now, we just create a basic structure based on the data

	// Process auth scheme
	schemeData, ok := data["auth_scheme"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("auth_scheme not found or invalid")
	}

	var scheme AuthScheme
	schemeTypeStr, ok := schemeData["type"].(string)
	if !ok {
		return fmt.Errorf("auth scheme type not found")
	}

	schemeType := AuthSchemeType(schemeTypeStr)

	// Process based on scheme type
	switch schemeType {
	case OpenIDConnectScheme:
		oidcScheme := &OpenIDConnectWithConfig{
			Type: schemeType,
		}
		if endpoint, ok := schemeData["authorization_endpoint"].(string); ok {
			oidcScheme.AuthorizationEndpoint = endpoint
		}
		if endpoint, ok := schemeData["token_endpoint"].(string); ok {
			oidcScheme.TokenEndpoint = endpoint
		}
		scheme = oidcScheme
	default:
		securityScheme := &SecurityScheme{
			Type: schemeType,
		}
		// Set other fields based on data if needed
		scheme = securityScheme
	}

	// Process credentials
	var rawCred, exchangedCred *AuthCredential

	if rawCredData, ok := data["raw_auth_credential"].(map[string]interface{}); ok {
		rawCred = &AuthCredential{}
		if authTypeStr, ok := rawCredData["auth_type"].(string); ok {
			rawCred.AuthType = AuthCredentialType(authTypeStr)
		}
		// Process other credential fields as needed
	}

	if exchangedCredData, ok := data["exchanged_auth_credential"].(map[string]interface{}); ok {
		exchangedCred = &AuthCredential{}
		if authTypeStr, ok := exchangedCredData["auth_type"].(string); ok {
			exchangedCred.AuthType = AuthCredentialType(authTypeStr)
		}
		// Process other credential fields as needed
	}

	// Set the config fields
	config.AuthScheme = scheme
	config.RawAuthCredential = rawCred
	config.ExchangedAuthCredential = exchangedCred

	return nil
}

// Default request processor instance
var RequestProcessor = &AuthLLMRequestProcessor{}
