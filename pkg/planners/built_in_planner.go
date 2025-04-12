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

package planners

import (
	"github.com/nvcnvn/adk-golang/pkg/agents"
	"github.com/nvcnvn/adk-golang/pkg/models"
)

// BuiltInPlanner is a planner that uses model's built-in thinking features.
// An error will be returned if this planner is used with models that don't
// support thinking.
type BuiltInPlanner struct {
	// ThinkingConfig contains configuration for model thinking features
	ThinkingConfig *models.ThinkingConfig
}

// NewBuiltInPlanner creates a new built-in planner with the specified thinking configuration.
func NewBuiltInPlanner(thinkingConfig *models.ThinkingConfig) *BuiltInPlanner {
	return &BuiltInPlanner{
		ThinkingConfig: thinkingConfig,
	}
}

// ApplyThinkingConfig applies the thinking config to the LLM request.
func (p *BuiltInPlanner) ApplyThinkingConfig(request *models.LlmRequest) {
	if p.ThinkingConfig != nil {
		// Set thinking config to the request when the models package supports it
		// This might need to be updated once the ThinkingConfig field is added to LlmRequest
	}
}

// BuildPlanningInstruction implements the Planner interface.
// For BuiltInPlanner, no additional instruction is needed as the thinking is handled
// by the model's built-in capabilities.
func (p *BuiltInPlanner) BuildPlanningInstruction(
	context agents.ReadonlyContext,
	request *models.LlmRequest,
) string {
	return ""
}

// ProcessPlanningResponse implements the Planner interface.
// For BuiltInPlanner, no additional processing is needed as the thinking is handled
// by the model's built-in capabilities.
func (p *BuiltInPlanner) ProcessPlanningResponse(
	context agents.CallbackContext,
	responseParts []*models.Part,
) []*models.Part {
	return responseParts
}
