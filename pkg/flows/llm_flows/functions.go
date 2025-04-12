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

package llm_flows

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/nvcnvn/adk-golang/pkg/agents"
	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/models"
	"github.com/nvcnvn/adk-golang/pkg/tools"
)

// HandleFunctionCalls processes function calls from the model response
func HandleFunctionCalls(ctx context.Context, invocationContext *agents.InvocationContext, functionCallEvent *events.Event, toolsDict map[string]*models.Tool) (*events.Event, error) {
	functionCalls := functionCallEvent.GetFunctionCalls()
	if len(functionCalls) == 0 {
		return nil, nil
	}

	// Create a response event
	functionResponseEvent := events.NewEvent()
	functionResponseEvent.InvocationID = invocationContext.InvocationID
	functionResponseEvent.Author = invocationContext.Agent.Name()

	// Process each function call
	content := &models.Content{
		Parts: make([]*models.Part, 0, len(functionCalls)),
	}

	for _, functionCall := range functionCalls {
		// Check if the tool exists in the dictionary
		_, exists := toolsDict[functionCall.Name]
		if !exists {
			log.Printf("Tool not found: %s", functionCall.Name)
			part := &models.Part{
				FunctionResponse: &models.FunctionResponse{
					Name:    functionCall.Name,
					Content: fmt.Sprintf("Error: Tool %s not found", functionCall.Name),
				},
			}
			content.Parts = append(content.Parts, part)
			continue
		}

		// Find the actual tool in the agent's canonical tools
		llmAgent, ok := invocationContext.Agent.(*agents.LlmAgent)
		if !ok {
			log.Printf("Agent is not an LLM agent")
			continue
		}

		var toolAdaptor *tools.LlmToolAdaptor
		for _, t := range llmAgent.CanonicalTools {
			if adaptor, ok := t.(*tools.LlmToolAdaptor); ok && adaptor.Name() == functionCall.Name {
				toolAdaptor = adaptor
				break
			}
		}

		if toolAdaptor == nil {
			log.Printf("LlmToolAdaptor for %s not found", functionCall.Name)
			part := &models.Part{
				FunctionResponse: &models.FunctionResponse{
					Name:    functionCall.Name,
					Content: fmt.Sprintf("Error: Tool adaptor for %s not found", functionCall.Name),
				},
			}
			content.Parts = append(content.Parts, part)
			continue
		}

		// Create a tool context
		toolContext := &tools.ToolContext{
			InvocationContext: invocationContext,
			EventActions:      functionResponseEvent.Actions,
		}

		// Execute the tool
		response, err := toolAdaptor.ExecuteFunctionCall(ctx, toolContext, functionCall)
		if err != nil {
			log.Printf("Error executing tool %s: %v", functionCall.Name, err)
			part := &models.Part{
				FunctionResponse: &models.FunctionResponse{
					Name:    functionCall.Name,
					Content: fmt.Sprintf("Error executing tool: %v", err),
				},
			}
			content.Parts = append(content.Parts, part)
			continue
		}

		// Add the response to the content
		part := &models.Part{
			FunctionResponse: &models.FunctionResponse{
				Name:    functionCall.Name,
				Content: response,
			},
		}
		if functionCall.ID != "" {
			part.FunctionResponse.ID = functionCall.ID
		}
		content.Parts = append(content.Parts, part)
	}

	functionResponseEvent.Content = content
	return functionResponseEvent, nil
}

// PopulateClientFunctionCallID generates client-side IDs for function calls
func PopulateClientFunctionCallID(event *events.Event) {
	functionCalls := event.GetFunctionCalls()
	for _, functionCall := range functionCalls {
		if functionCall.ID == "" {
			functionCall.ID = uuid.New().String()
		}
	}
}

// GetLongRunningFunctionCalls identifies which function calls are long-running
func GetLongRunningFunctionCalls(functionCalls []*models.FunctionCall, toolsDict map[string]*tools.LlmToolAdaptor) []string {
	longRunningToolIDs := make([]string, 0)

	for _, functionCall := range functionCalls {
		if tool, exists := toolsDict[functionCall.Name]; exists && tool.IsLongRunning() {
			longRunningToolIDs = append(longRunningToolIDs, functionCall.ID)
		}
	}

	return longRunningToolIDs
}

// GenerateAuthEvent creates an auth event if needed
func GenerateAuthEvent(invocationContext *agents.InvocationContext, functionResponseEvent *events.Event) *events.Event {
	// Check if any function responses require authentication
	for _, part := range functionResponseEvent.Content.Parts {
		if part.FunctionResponse != nil && part.FunctionResponse.AuthRequest != nil {
			// Create an auth event
			authEvent := events.NewEvent()
			authEvent.InvocationID = invocationContext.InvocationID
			authEvent.Author = invocationContext.Agent.Name()
			authEvent.Content = &models.Content{
				Parts: []*models.Part{
					{
						AuthRequest: part.FunctionResponse.AuthRequest,
					},
				},
			}
			return authEvent
		}
	}
	return nil
}
