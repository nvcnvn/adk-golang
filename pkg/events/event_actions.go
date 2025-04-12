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

package events

import (
	"github.com/nvcnvn/adk-golang/pkg/auth"
)

// EventActions represents the actions attached to an event.
type EventActions struct {
	// SkipSummarization, if true, means the agent won't call model to summarize function response.
	// Only used for function_response events.
	SkipSummarization bool `json:"skip_summarization,omitempty"`

	// StateDelta indicates that the event is updating the state with the given delta.
	StateDelta map[string]interface{} `json:"state_delta,omitempty"`

	// ArtifactDelta indicates that the event is updating an artifact.
	// Key is the filename, value is the version.
	ArtifactDelta map[string]int `json:"artifact_delta,omitempty"`

	// TransferToAgent, if set, indicates the event transfers to the specified agent.
	TransferToAgent string `json:"transfer_to_agent,omitempty"`

	// Escalate indicates the agent is escalating to a higher level agent.
	Escalate bool `json:"escalate,omitempty"`

	// RequestedAuthConfigs will only be set by a tool response indicating tool request euc.
	// Map key is the function call ID since one function call response (from model)
	// could correspond to multiple function calls.
	// Map value is the required auth config.
	RequestedAuthConfigs map[string]*auth.AuthConfig `json:"requested_auth_configs,omitempty"`
}

// NewEventActions creates a new EventActions with default values.
func NewEventActions() *EventActions {
	return &EventActions{
		StateDelta:           make(map[string]interface{}),
		ArtifactDelta:        make(map[string]int),
		RequestedAuthConfigs: make(map[string]*auth.AuthConfig),
	}
}

// Update applies the state delta from another EventActions.
func (a *EventActions) Update(other *EventActions) {
	if other == nil {
		return
	}

	if other.SkipSummarization {
		a.SkipSummarization = true
	}

	if other.TransferToAgent != "" {
		a.TransferToAgent = other.TransferToAgent
	}

	if other.Escalate {
		a.Escalate = true
	}

	// Merge state deltas
	for k, v := range other.StateDelta {
		a.StateDelta[k] = v
	}

	// Merge artifact deltas
	for k, v := range other.ArtifactDelta {
		a.ArtifactDelta[k] = v
	}

	// Merge requested auth configs
	for k, v := range other.RequestedAuthConfigs {
		a.RequestedAuthConfigs[k] = v
	}
}
