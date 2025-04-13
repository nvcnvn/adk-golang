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

package agents

import (
	"fmt"
	"log"
	"math"
)

// StreamingMode defines the different streaming modes available
type StreamingMode string

const (
	// StreamingModeNone indicates no streaming
	StreamingModeNone StreamingMode = ""

	// StreamingModeSSE indicates Server-Sent Events streaming
	StreamingModeSSE StreamingMode = "sse"

	// StreamingModeBIDI indicates bidirectional streaming
	StreamingModeBIDI StreamingMode = "bidi"
)

// RunConfig configures the runtime behavior of agents
type RunConfig struct {
	// MaxLlmCalls is the maximum number of LLM API calls to allow for a single run
	// Used to prevent infinite loops or excessive API usage
	MaxLlmCalls int `json:"maxLlmCalls"`

	// StreamingMode configures how streaming responses are handled
	StreamingMode StreamingMode `json:"streamingMode"`

	// AllowStateChangesOnStreaming controls whether state can change during streaming
	AllowStateChangesOnStreaming bool `json:"allowStateChangesOnStreaming"`

	// Debug enables additional debug information
	Debug bool `json:"debug"`
}

// NewRunConfig creates a new RunConfig with default values
func NewRunConfig() *RunConfig {
	return &RunConfig{
		MaxLlmCalls:                  10,
		StreamingMode:                StreamingModeNone,
		AllowStateChangesOnStreaming: false,
		Debug:                        false,
	}
}

// Validate validates the configuration and returns an error if invalid
func (c *RunConfig) Validate() error {
	if c.MaxLlmCalls == math.MaxInt {
		return fmt.Errorf("maxLlmCalls should be less than system max int")
	}

	if c.MaxLlmCalls <= 0 {
		log.Printf("WARNING: maxLlmCalls is less than or equal to 0. This will result in " +
			"no enforcement on total number of llm calls that will be made for a " +
			"run. This may not be ideal, as this could result in a never " +
			"ending communication between the model and the agent in certain cases.")
	}

	switch c.StreamingMode {
	case StreamingModeNone, StreamingModeSSE, StreamingModeBIDI:
		// Valid modes
	default:
		return fmt.Errorf("invalid streaming mode: %s", c.StreamingMode)
	}

	return nil
}

// WithMaxLlmCalls sets the maximum number of LLM calls
func (c *RunConfig) WithMaxLlmCalls(max int) *RunConfig {
	c.MaxLlmCalls = max
	return c
}

// WithStreamingMode sets the streaming mode
func (c *RunConfig) WithStreamingMode(mode StreamingMode) *RunConfig {
	c.StreamingMode = mode
	return c
}

// WithAllowStateChangesOnStreaming sets whether state changes are allowed during streaming
func (c *RunConfig) WithAllowStateChangesOnStreaming(allow bool) *RunConfig {
	c.AllowStateChangesOnStreaming = allow
	return c
}

// WithDebug sets the debug mode
func (c *RunConfig) WithDebug(debug bool) *RunConfig {
	c.Debug = debug
	return c
}
