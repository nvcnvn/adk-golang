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

// Package planners contains planner interfaces and implementations for ADK.
package planners

import (
	"github.com/nvcnvn/adk-golang/pkg/agents"
	"github.com/nvcnvn/adk-golang/pkg/models"
)

// Planner is the interface for all planners.
// A planner allows the agent to generate plans for the queries to guide its action.
type Planner interface {
	// BuildPlanningInstruction builds the system instruction to be appended to the LLM request for planning.
	BuildPlanningInstruction(
		context agents.ReadonlyContext,
		request *models.LlmRequest,
	) string

	// ProcessPlanningResponse processes the LLM response for planning.
	ProcessPlanningResponse(
		context agents.CallbackContext,
		responseParts []*models.Part,
	) []*models.Part
}
