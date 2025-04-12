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

package models

import (
	"context"
	"fmt"
)

// UnifiedModelFactory provides a unified way to create models, using both the
// standard ModelRegistry and the enhanced EnhancedRegistry with regex support.
type UnifiedModelFactory struct {
	standardRegistry *ModelRegistry
	enhancedRegistry *EnhancedRegistry
}

var unifiedFactory *UnifiedModelFactory

// GetUnifiedModelFactory returns a singleton instance of UnifiedModelFactory.
func GetUnifiedModelFactory() *UnifiedModelFactory {
	if unifiedFactory == nil {
		unifiedFactory = &UnifiedModelFactory{
			standardRegistry: GetRegistry(),
			enhancedRegistry: GetEnhancedRegistry(),
		}
	}
	return unifiedFactory
}

// GetModel tries to get a model from both registries, prioritizing the standard registry.
func (f *UnifiedModelFactory) GetModel(modelName string) (Model, error) {
	// First try the standard registry
	if model, ok := f.standardRegistry.Get(modelName); ok {
		return model, nil
	}

	// If not found in standard registry, try the enhanced registry
	model, err := f.enhancedRegistry.GetModel(modelName)
	if err != nil {
		return nil, fmt.Errorf("model %s not found in any registry: %w", modelName, err)
	}

	return model, nil
}

// GetLLM creates a new LLM based on the model name.
// This is a bridge between the old Model interface and the new LLM interface.
func (f *UnifiedModelFactory) GetLLM(modelName string) (LLM, error) {
	// Try to get a Model and wrap it
	model, err := f.GetModel(modelName)
	if err != nil {
		return nil, err
	}

	// Create an adapter from Model to LLM
	return &ModelToLLMAdapter{model: model}, nil
}

// LLMTypeFactory is a function that creates a new LLM instance.
type LLMTypeFactory func(modelName string) LLM

// ModelToLLMAdapter adapts the Model interface to the LLM interface.
type ModelToLLMAdapter struct {
	model Model
}

// SupportedModels returns a list with only the exact model name.
func (a *ModelToLLMAdapter) SupportedModels() []string {
	return []string{a.model.Name()}
}

// GenerateContent generates content by using the underlying Model.
func (a *ModelToLLMAdapter) GenerateContent(ctx context.Context, request *LlmRequest) (*LlmResponse, error) {
	// Convert request to messages for the underlying Model
	messages := []Message{}

	// Add system instructions as a message if present
	if request.SystemInstructions != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: request.SystemInstructions,
		})
	}

	// Add content parts as messages
	if request.Contents != nil && len(request.Contents.Parts) > 0 {
		for _, part := range request.Contents.Parts {
			role := "user"
			if part.Role != "" {
				role = part.Role
			}

			messages = append(messages, Message{
				Role:    role,
				Content: part.Text,
			})
		}
	}

	// If no messages were added, add a default one
	if len(messages) == 0 {
		messages = append(messages, Message{
			Role:    "user",
			Content: "Hello",
		})
	}

	// Call the underlying Model
	text, err := a.model.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}

	// Convert the response to LlmResponse format
	response := &LlmResponse{
		Content: &Content{
			Parts: []*Part{{
				Text: text,
				Role: "assistant",
			}},
		},
	}

	return response, nil
}

// GenerateContentStream adapts the streaming interface.
func (a *ModelToLLMAdapter) GenerateContentStream(ctx context.Context, request *LlmRequest) (<-chan *LlmResponse, error) {
	// Convert request to messages for the underlying Model
	messages := []Message{}

	// Add system instructions as a message if present
	if request.SystemInstructions != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: request.SystemInstructions,
		})
	}

	// Add content parts as messages
	if request.Contents != nil && len(request.Contents.Parts) > 0 {
		for _, part := range request.Contents.Parts {
			role := "user"
			if part.Role != "" {
				role = part.Role
			}

			messages = append(messages, Message{
				Role:    role,
				Content: part.Text,
			})
		}
	}

	// If no messages were added, add a default one
	if len(messages) == 0 {
		messages = append(messages, Message{
			Role:    "user",
			Content: "Hello",
		})
	}

	// Call the streaming method on the Model
	streamChan, err := a.model.GenerateStream(ctx, messages)
	if err != nil {
		return nil, err
	}

	// Convert the stream to LlmResponse channel
	responseChan := make(chan *LlmResponse)

	go func() {
		defer close(responseChan)

		for streamResp := range streamChan {
			if streamResp.Error != nil {
				responseChan <- &LlmResponse{
					Content: &Content{
						Parts: []*Part{{
							Text: fmt.Sprintf("Error: %v", streamResp.Error),
							Role: "assistant",
						}},
					},
					ErrorMessage: streamResp.Error.Error(),
				}
				return
			}

			responseChan <- &LlmResponse{
				Content: &Content{
					Parts: []*Part{{
						Text: streamResp.Content,
						Role: "assistant",
					}},
				},
				Partial: !streamResp.Done,
			}
		}
	}()

	return responseChan, nil
}

// Connect returns an error as this adapter doesn't support real-time connections.
func (a *ModelToLLMAdapter) Connect(ctx context.Context, request *LlmRequest) (LlmConnection, error) {
	return nil, fmt.Errorf("real-time connection not supported by model %s", a.model.Name())
}
