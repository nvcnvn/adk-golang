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

package sessions

import (
	"sync"
)

const (
	// AppPrefix for app-specific state values
	AppPrefix = "app:"
	// UserPrefix for user-specific state values
	UserPrefix = "user:"
	// TempPrefix for temporary state values
	TempPrefix = "temp:"
)

// State represents a session state with delta tracking
type State struct {
	value map[string]interface{} // current state values
	delta map[string]interface{} // pending changes
	mu    sync.RWMutex
}

// NewState creates a new State instance
func NewState(value map[string]interface{}, delta map[string]interface{}) *State {
	if value == nil {
		value = make(map[string]interface{})
	}
	if delta == nil {
		delta = make(map[string]interface{})
	}
	return &State{
		value: value,
		delta: delta,
	}
}

// Get retrieves a state value by key
func (s *State) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check delta first
	if val, ok := s.delta[key]; ok {
		return val, true
	}

	// Then check base value
	val, ok := s.value[key]
	return val, ok
}

// GetOrDefault retrieves a state value or returns the default if not found
func (s *State) GetOrDefault(key string, defaultVal interface{}) interface{} {
	val, ok := s.Get(key)
	if !ok {
		return defaultVal
	}
	return val
}

// Set sets a value in both current state and delta
func (s *State) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.value[key] = value
	s.delta[key] = value
}

// HasDelta returns true if there are pending changes
func (s *State) HasDelta() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.delta) > 0
}

// Contains checks if a key exists in the state
func (s *State) Contains(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, inDelta := s.delta[key]
	_, inValue := s.value[key]
	return inDelta || inValue
}

// Update applies changes from a delta map to the state
func (s *State) Update(delta map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range delta {
		s.value[k] = v
		s.delta[k] = v
	}
}

// ToMap returns the combined state as a map
func (s *State) ToMap() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range s.value {
		result[k] = v
	}
	for k, v := range s.delta {
		result[k] = v
	}
	return result
}

// ClearDelta clears the pending changes
func (s *State) ClearDelta() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.delta = make(map[string]interface{})
}
