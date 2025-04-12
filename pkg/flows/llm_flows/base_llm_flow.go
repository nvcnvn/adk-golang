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

	"github.com/nvcnvn/adk-golang/pkg/agents"
	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/models"
	"github.com/nvcnvn/adk-golang/pkg/tools"
	"github.com/nvcnvn/adk-golang/pkg/types"
)

// LlmRequestProcessor defines an interface for processing LLM requests before they are sent
type LlmRequestProcessor interface {
	Run(ctx context.Context, invocationContext *agents.InvocationContext, llmRequest *models.LlmRequest) (<-chan *events.Event, error)
}

// LlmResponseProcessor defines an interface for processing LLM responses after they are received
type LlmResponseProcessor interface {
	Run(ctx context.Context, invocationContext *agents.InvocationContext, llmResponse *models.LlmResponse) (<-chan *events.Event, error)
}

// BaseLlmFlow represents a flow that handles LLM interactions
type BaseLlmFlow struct {
	RequestProcessors  []LlmRequestProcessor
	ResponseProcessors []LlmResponseProcessor
}

// NewBaseLlmFlow creates a new BaseLlmFlow instance
func NewBaseLlmFlow() *BaseLlmFlow {
	return &BaseLlmFlow{
		RequestProcessors:  make([]LlmRequestProcessor, 0),
		ResponseProcessors: make([]LlmResponseProcessor, 0),
	}
}

// Run executes the flow with the given invocation context
func (f *BaseLlmFlow) Run(ctx context.Context, invocationContext *agents.InvocationContext) (<-chan *events.Event, error) {
	eventCh := make(chan *events.Event)

	go func() {
		defer close(eventCh)

		for {
			responseCh, err := f.runOneStep(ctx, invocationContext)
			if err != nil {
				log.Printf("Error running flow step: %v", err)
				return
			}

			var lastEvent *events.Event
			for event := range responseCh {
				lastEvent = event
				eventCh <- event
			}

			if lastEvent == nil || lastEvent.IsFinalResponse() {
				break
			}
		}
	}()

	return eventCh, nil
}

// runOneStep executes one step of the flow (one LLM call)
func (f *BaseLlmFlow) runOneStep(ctx context.Context, invocationContext *agents.InvocationContext) (<-chan *events.Event, error) {
	eventCh := make(chan *events.Event)

	go func() {
		defer close(eventCh)

		// Create a new LLM request
		llmRequest := &models.LlmRequest{}

		// Preprocess the request
		preprocessCh, err := f.preprocess(ctx, invocationContext, llmRequest)
		if err != nil {
			log.Printf("Error preprocessing request: %v", err)
			return
		}

		for event := range preprocessCh {
			eventCh <- event
		}

		if invocationContext.EndInvocation {
			return
		}

		// Create a new event for the model response
		modelResponseEvent := events.NewEvent()
		modelResponseEvent.InvocationID = invocationContext.InvocationID
		modelResponseEvent.Author = invocationContext.Agent.Name()

		// Call the LLM
		responseCh, err := f.callLLM(ctx, invocationContext, llmRequest, modelResponseEvent)
		if err != nil {
			log.Printf("Error calling LLM: %v", err)
			return
		}

		for llmResponse := range responseCh {
			// Postprocess the response
			postprocessCh, err := f.postprocess(ctx, invocationContext, llmRequest, llmResponse, modelResponseEvent)
			if err != nil {
				log.Printf("Error postprocessing response: %v", err)
				return
			}

			for event := range postprocessCh {
				eventCh <- event
			}
		}
	}()

	return eventCh, nil
}

// preprocess runs request processors before calling the LLM
func (f *BaseLlmFlow) preprocess(ctx context.Context, invocationContext *agents.InvocationContext, llmRequest *models.LlmRequest) (<-chan *events.Event, error) {
	eventCh := make(chan *events.Event)

	go func() {
		defer close(eventCh)

		llmAgent, ok := invocationContext.Agent.(*agents.LlmAgent)
		if !ok {
			return
		}

		// Run request processors
		for _, processor := range f.RequestProcessors {
			processorCh, err := processor.Run(ctx, invocationContext, llmRequest)
			if err != nil {
				log.Printf("Error running request processor: %v", err)
				return
			}

			for event := range processorCh {
				eventCh <- event
			}
		}

		// Process tools using LlmToolAdaptor
		for _, tool := range llmAgent.CanonicalTools {
			if adaptor, ok := tool.(*tools.LlmToolAdaptor); ok {
				toolCtx := &tools.ToolContext{
					InvocationContext: invocationContext,
				}
				if err := adaptor.ProcessLlmRequest(ctx, toolCtx, llmRequest); err != nil {
					log.Printf("Error processing tool request: %v", err)
					continue
				}
			}
		}
	}()

	return eventCh, nil
}

// callLLM calls the LLM with the given request
func (f *BaseLlmFlow) callLLM(ctx context.Context, invocationContext *agents.InvocationContext, llmRequest *models.LlmRequest, modelResponseEvent *events.Event) (<-chan *models.LlmResponse, error) {
	responseCh := make(chan *models.LlmResponse)

	go func() {
		defer close(responseCh)

		llmAgent, ok := invocationContext.Agent.(*agents.LlmAgent)
		if !ok {
			return
		}

		// Handle before model callback if exists
		if llmAgent.BeforeModelCallback != nil {
			callbackCtx := &agents.CallbackContext{
				InvocationContext: invocationContext,
				EventActions:      modelResponseEvent.Actions,
			}

			response := llmAgent.BeforeModelCallback(callbackCtx, llmRequest)
			if response != nil {
				responseCh <- response
				return
			}
		}

		// Increment LLM call count
		invocationContext.IncrementLlmCallCount()

		// Get the canonical model
		llm := llmAgent.CanonicalModel

		// Determine if streaming is requested
		streaming := invocationContext.RunConfig.StreamingMode == types.StreamingModeSSE

		var err error
		if streaming {
			// If streaming, use the streaming generate function
			var llmResponseCh <-chan *models.LlmResponse
			llmResponseCh, err = llm.GenerateContentStream(ctx, llmRequest)
			if err != nil {
				log.Printf("Error generating streaming content: %v", err)
				return
			}

			// Forward each streamed response through our channel
			for llmResponse := range llmResponseCh {
				// Handle after model callback if exists
				if llmAgent.AfterModelCallback != nil {
					callbackCtx := &agents.CallbackContext{
						InvocationContext: invocationContext,
						EventActions:      modelResponseEvent.Actions,
					}

					alteredResponse := llmAgent.AfterModelCallback(callbackCtx, llmResponse)
					if alteredResponse != nil {
						llmResponse = alteredResponse
					}
				}

				responseCh <- llmResponse
			}
		} else {
			// If not streaming, use the non-streaming generate function
			var llmResponse *models.LlmResponse
			llmResponse, err = llm.GenerateContent(ctx, llmRequest)
			if err != nil {
				log.Printf("Error generating content: %v", err)
				return
			}

			// Handle after model callback if exists
			if llmAgent.AfterModelCallback != nil {
				callbackCtx := &agents.CallbackContext{
					InvocationContext: invocationContext,
					EventActions:      modelResponseEvent.Actions,
				}

				alteredResponse := llmAgent.AfterModelCallback(callbackCtx, llmResponse)
				if alteredResponse != nil {
					llmResponse = alteredResponse
				}
			}

			// Send the single response through our channel
			responseCh <- llmResponse
		}
	}()

	return responseCh, nil
}

// postprocess runs response processors after calling the LLM
func (f *BaseLlmFlow) postprocess(ctx context.Context, invocationContext *agents.InvocationContext, llmRequest *models.LlmRequest, llmResponse *models.LlmResponse, modelResponseEvent *events.Event) (<-chan *events.Event, error) {
	eventCh := make(chan *events.Event)

	go func() {
		defer close(eventCh)

		// Run response processors
		for _, processor := range f.ResponseProcessors {
			processorCh, err := processor.Run(ctx, invocationContext, llmResponse)
			if err != nil {
				log.Printf("Error running response processor: %v", err)
				continue
			}

			for event := range processorCh {
				eventCh <- event
			}
		}

		// Skip if no content and no error code
		if llmResponse.Content == nil && llmResponse.ErrorCode == "" && !llmResponse.Interrupted {
			return
		}

		// Finalize the model response event
		finalEvent := f.finalizeModelResponseEvent(llmRequest, llmResponse, modelResponseEvent)
		eventCh <- finalEvent

		// Handle function calls if any
		functionCalls := finalEvent.GetFunctionCalls()
		if len(functionCalls) > 0 {
			// Process function calls (implementation would be in the functions package)
			functionResponseEvent, err := HandleFunctionCalls(ctx, invocationContext, finalEvent, llmRequest.ToolsDict)
			if err != nil {
				log.Printf("Error handling function calls: %v", err)
				return
			}

			if functionResponseEvent != nil {
				eventCh <- functionResponseEvent

				// Handle agent transfer if needed
				if functionResponseEvent.Actions.TransferToAgent != "" {
					agentToRun, err := f.getAgentToRun(invocationContext, functionResponseEvent.Actions.TransferToAgent)
					if err != nil {
						log.Printf("Error finding agent to transfer to: %v", err)
						return
					}

					transferCh, err := agentToRun.Run(ctx, invocationContext)
					if err != nil {
						log.Printf("Error running transferred agent: %v", err)
						return
					}

					for event := range transferCh {
						eventCh <- event
					}
				}
			}
		}
	}()

	return eventCh, nil
}

// finalizeModelResponseEvent combines the model response with the event
func (f *BaseLlmFlow) finalizeModelResponseEvent(llmRequest *models.LlmRequest, llmResponse *models.LlmResponse, modelResponseEvent *events.Event) *events.Event {
	// Copy properties from LLM response to the event
	if llmResponse.Content != nil {
		modelResponseEvent.Content = llmResponse.Content
	}
	modelResponseEvent.Partial = llmResponse.Partial
	modelResponseEvent.ErrorCode = llmResponse.ErrorCode
	modelResponseEvent.ErrorMessage = llmResponse.ErrorMessage
	modelResponseEvent.Interrupted = llmResponse.Interrupted

	// Process function calls if present
	if modelResponseEvent.Content != nil && len(modelResponseEvent.GetFunctionCalls()) > 0 {
		PopulateClientFunctionCallID(modelResponseEvent)

		// Convert toolsDict from models.Tool to tools.LlmToolAdaptor
		toolAdaptors := make(map[string]*tools.LlmToolAdaptor)
		for name, tool := range llmRequest.ToolsDict {
			// Create a new adaptor with the proper settings
			baseTool := tools.NewTool(name, "", tools.ToolSchema{}, nil)
			toolAdaptors[name] = tools.NewLlmToolAdaptor(baseTool, tool.IsLongRunning)
		}

		modelResponseEvent.LongRunningToolIDs = GetLongRunningFunctionCalls(modelResponseEvent.GetFunctionCalls(), toolAdaptors)
	}

	return modelResponseEvent
}

// getAgentToRun finds the agent to transfer to
func (f *BaseLlmFlow) getAgentToRun(invocationContext *agents.InvocationContext, transferToAgent string) (agents.BaseAgent, error) {
	rootAgent := invocationContext.Agent.RootAgent()
	agentToRun := rootAgent.FindAgent(transferToAgent)
	if agentToRun == nil {
		return nil, fmt.Errorf("agent %s not found in the agent tree", transferToAgent)
	}
	return agentToRun, nil
}
