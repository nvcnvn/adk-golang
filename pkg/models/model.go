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

// Package models provides interfaces and implementations for language models.
package models

import (
	"context"
	"errors"
	"sync"
)

// Message represents a message in a conversation with a model.
type Message struct {
	Role    string            // Role can be "user", "system", or "assistant"
	Content string            // The text content of the message
	Attrs   map[string]string // Additional attributes for the message
}

// Model is the interface for language models.
type Model interface {
	// Name returns the name of the model.
	Name() string

	// Generate generates a response to the given messages.
	Generate(ctx context.Context, messages []Message) (string, error)

	// GenerateStream generates a streaming response to the given messages.
	GenerateStream(ctx context.Context, messages []Message) (chan StreamedResponse, error)
}

// StreamedResponse represents a partial response from a streaming model.
type StreamedResponse struct {
	Content string
	Error   error
	Done    bool
}

// ModelRegistry keeps track of available models.
type ModelRegistry struct {
	models map[string]Model
	mu     sync.RWMutex
}

var (
	registry     = &ModelRegistry{models: make(map[string]Model)}
	registryOnce sync.Once
)

// GetRegistry returns the singleton model registry.
func GetRegistry() *ModelRegistry {
	registryOnce.Do(func() {
		registry = &ModelRegistry{
			models: make(map[string]Model),
		}
	})
	return registry
}

// Register registers a model with the registry.
func (r *ModelRegistry) Register(model Model) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[model.Name()] = model
}

// Get returns a model from the registry by name.
func (r *ModelRegistry) Get(name string) (Model, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	model, ok := r.models[name]
	return model, ok
}

// List returns all registered models.
func (r *ModelRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var names []string
	for name := range r.models {
		names = append(names, name)
	}
	return names
}

// BaseModel provides a common implementation of the Model interface.
type BaseModel struct {
	name string
}

// Name returns the name of the model.
func (m *BaseModel) Name() string {
	return m.name
}

// Generate generates a response to the given messages.
func (m *BaseModel) Generate(ctx context.Context, messages []Message) (string, error) {
	return "", errors.New("not implemented")
}

// GenerateStream generates a streaming response to the given messages.
func (m *BaseModel) GenerateStream(ctx context.Context, messages []Message) (chan StreamedResponse, error) {
	return nil, errors.New("not implemented")
}

// MockModel is a simple model implementation for testing.
type MockModel struct {
	BaseModel
	response string
}

// NewMockModel creates a new MockModel with the given name and fixed response.
func NewMockModel(name, response string) *MockModel {
	return &MockModel{
		BaseModel: BaseModel{name: name},
		response:  response,
	}
}

// Generate returns the fixed response.
func (m *MockModel) Generate(ctx context.Context, messages []Message) (string, error) {
	return m.response, nil
}

// GenerateStream streams the fixed response character by character.
func (m *MockModel) GenerateStream(ctx context.Context, messages []Message) (chan StreamedResponse, error) {
	ch := make(chan StreamedResponse)

	go func() {
		defer close(ch)

		for i, char := range m.response {
			select {
			case <-ctx.Done():
				ch <- StreamedResponse{
					Error: ctx.Err(),
					Done:  true,
				}
				return
			default:
				ch <- StreamedResponse{
					Content: string(char),
					Done:    i == len(m.response)-1,
				}
			}
		}
	}()

	return ch, nil
}
