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
}

// NewAuthLLMRequestProcessor creates a new AuthLLMRequestProcessor
func NewAuthLLMRequestProcessor(handleFunctionCalls FunctionCallsHandler) *AuthLLMRequestProcessor {
	return &AuthLLMRequestProcessor{
		HandleFunctionCalls: handleFunctionCalls,
	}
}

// ProcessAsync processes an LLM request asynchronously
func (p *AuthLLMRequestProcessor) ProcessAsync(ctx context.Context, invocation InvocationContext, llmRequest LLMRequest) ([]Event, error) {
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
			
			// Initialize auth handler with the config
			// In a real implementation, we would properly parse the auth config
			// For now, we just simulate that part
			authConfig := AuthConfig{
				// In a real implementation, this would be properly deserialized
			}
			
			handler := NewAuthHandler(authConfig)
			err := handler.ParseAndStoreAuthResponse(session)
			if err != nil {
				return nil, err
			}
		}
		
		// Found a user event, no need to look further
		break
	}
	
	if len(requestEUCFunctionCallIDs) == 0 {
		return nil, nil
	}
	
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
			args, ok := functionCall.Args()["function_call_id"]
			if !ok {
				continue
			}
			
			functionCallID, ok := args.(string)
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
			canonicalTools := make(map[string]interface{})
			
			// In a real implementation, we would get the tools from the agent
			// For now, we just simulate that part
			
			for _, functionCall := range functionCalls {
				if _, ok := toolsToResume[functionCall.ID()]; ok {
					if p.HandleFunctionCalls != nil {
						functionResponseEvent, err := p.HandleFunctionCalls(
							ctx, 
							invocation, 
							event, 
							canonicalTools, 
							toolsToResume,
						)
						if err != nil {
							return nil, err
						}
						
						if functionResponseEvent != nil {
							return []Event{functionResponseEvent}, nil
						}
					}
				}
			}
			
			return nil, nil
		}
		
		return nil, nil
	}
	
	return nil, nil
}

// Default request processor instance
var RequestProcessor = &AuthLLMRequestProcessor{}