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

// ThinkingConfig holds configuration for model thinking features.
// These are advanced model capabilities that let models perform reasoning
// steps before producing a final response.
type ThinkingConfig struct {
	// Enabled indicates whether thinking should be enabled
	Enabled bool `json:"enabled,omitempty"`

	// Verbosity controls how detailed the thinking output should be
	Verbosity string `json:"verbosity,omitempty"`

	// Custom configuration parameters specific to certain models
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// NewThinkingConfig creates a new thinking configuration with default values.
func NewThinkingConfig() *ThinkingConfig {
	return &ThinkingConfig{
		Enabled:   true,
		Verbosity: "auto",
		Custom:    make(map[string]interface{}),
	}
}
