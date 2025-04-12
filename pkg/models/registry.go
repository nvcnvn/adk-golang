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
	"fmt"
	"log"
	"regexp"
	"sync"
)

// EnhancedModelFactory is a function that creates a model given a name.
type EnhancedModelFactory func(modelName string) (Model, error)

// RegistryEntry combines a regex pattern with its associated model factory.
type RegistryEntry struct {
	Pattern *regexp.Regexp
	Factory EnhancedModelFactory
}

// EnhancedRegistry is a registry for models that supports regex-based lookup.
type EnhancedRegistry struct {
	entries []RegistryEntry
	models  map[string]Model // Cache for already created models
	mu      sync.RWMutex
}

var (
	enhancedRegistry     *EnhancedRegistry
	enhancedRegistryOnce sync.Once
)

// GetEnhancedRegistry returns the singleton enhanced model registry.
func GetEnhancedRegistry() *EnhancedRegistry {
	enhancedRegistryOnce.Do(func() {
		enhancedRegistry = &EnhancedRegistry{
			entries: make([]RegistryEntry, 0),
			models:  make(map[string]Model),
		}
	})
	return enhancedRegistry
}

// RegisterPattern registers a model factory with a regex pattern.
func (r *EnhancedRegistry) RegisterPattern(pattern string, factory EnhancedModelFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern %s: %w", pattern, err)
	}

	r.entries = append(r.entries, RegistryEntry{
		Pattern: regex,
		Factory: factory,
	})

	log.Printf("Registered model pattern: %s", pattern)
	return nil
}

// GetModel returns or creates a model that matches the given name.
func (r *EnhancedRegistry) GetModel(name string) (Model, error) {
	// First check the cache
	r.mu.RLock()
	model, exists := r.models[name]
	r.mu.RUnlock()

	if exists {
		return model, nil
	}

	// Try to create a new model
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check again in case another goroutine created it
	if model, exists = r.models[name]; exists {
		return model, nil
	}

	// Find a factory for the model name
	for _, entry := range r.entries {
		if entry.Pattern.MatchString(name) {
			model, err := entry.Factory(name)
			if err != nil {
				return nil, err
			}

			// Cache the model
			r.models[name] = model
			return model, nil
		}
	}

	return nil, fmt.Errorf("no model factory found for model name: %s", name)
}

// ListPatterns returns all registered patterns.
func (r *EnhancedRegistry) ListPatterns() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	patterns := make([]string, 0, len(r.entries))
	for _, entry := range r.entries {
		patterns = append(patterns, entry.Pattern.String())
	}

	return patterns
}
