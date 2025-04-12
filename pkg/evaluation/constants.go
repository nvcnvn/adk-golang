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

package evaluation

// EvaluationConstants holds the key names used in evaluation data structures
const (
	Query           = "query"
	ExpectedToolUse = "expected_tool_use"
	Response        = "response"
	Reference       = "reference"
	ToolName        = "tool_name"
	ToolInput       = "tool_input"
	MockToolOutput  = "mock_tool_output"
	ActualToolUse   = "actual_tool_use"
)

// EvaluationCriteria defines constants for evaluation criteria
const (
	ToolTrajectoryScoreKey     = "tool_trajectory_avg_score"
	ResponseEvaluationScoreKey = "response_evaluation_score"
	ResponseMatchScoreKey      = "response_match_score"

	// Default settings
	DefaultToolTrajectoryScore = 1.0 // 1-point scale; 1.0 is perfect
	DefaultResponseMatchScore  = 0.8 // Rouge-1 text match; 0.8 is default
)

// AllowedCriteria defines the allowed evaluation criteria
var AllowedCriteria = []string{
	ToolTrajectoryScoreKey,
	ResponseEvaluationScoreKey,
	ResponseMatchScoreKey,
}

// DefaultCriteria defines the default evaluation criteria settings
var DefaultCriteria = map[string]float64{
	ToolTrajectoryScoreKey: DefaultToolTrajectoryScore,
	ResponseMatchScoreKey:  DefaultResponseMatchScore,
}
