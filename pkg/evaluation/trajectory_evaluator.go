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

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// TrajectoryEvaluator evaluates tool use trajectories for accuracy
type TrajectoryEvaluator struct{}

// EvaluationResult represents the result of a trajectory evaluation
type EvaluationResult struct {
	Query           string    `json:"query"`
	Response        string    `json:"response"`
	ActualToolUse   []ToolUse `json:"actual_tool_use"`
	ExpectedToolUse []ToolUse `json:"expected_tool_use"`
	ToolUseAccuracy float64   `json:"tool_use_accuracy"`
}

// FailureInfo represents details about a trajectory evaluation failure
type FailureInfo struct {
	Turn     int       `json:"turn"`
	Query    string    `json:"query"`
	Actual   []ToolUse `json:"actual"`
	Expected []ToolUse `json:"expected"`
}

// NewTrajectoryEvaluator creates a new trajectory evaluator
func NewTrajectoryEvaluator() *TrajectoryEvaluator {
	return &TrajectoryEvaluator{}
}

// Evaluate returns the mean tool use accuracy of the evaluation dataset
// Tool use accuracy is calculated by comparing the expected and actual tool
// use trajectories. An exact match scores 1, 0 otherwise. The final value is an
// average of these individual scores.
// Value range: [0, 1], where 0 means none of the tool use entries aligned,
// and 1 would mean all of them aligned. Higher value is better.
func (te *TrajectoryEvaluator) Evaluate(dataset EvaluationDataset, printDetailedResults bool) (float64, error) {
	if len(dataset) == 0 {
		return 0, fmt.Errorf("the evaluation dataset is empty")
	}

	var results []EvaluationResult
	var failures []FailureInfo

	for _, conversation := range dataset {
		for i, entry := range conversation {
			result, failure, err := te.evaluateEntry(entry)
			if err != nil {
				return 0, err
			}
			results = append(results, result)

			if failure != nil {
				failure.Turn = i + 1
				failures = append(failures, *failure)
			}
		}
	}

	// Report failures if any
	te.reportFailures(failures)

	// Print detailed results if requested
	if printDetailedResults {
		te.printResults(results)
	}

	// Calculate mean accuracy
	var totalAccuracy float64
	for _, result := range results {
		totalAccuracy += result.ToolUseAccuracy
	}

	if len(results) == 0 {
		return 0, nil
	}

	return totalAccuracy / float64(len(results)), nil
}

// evaluateEntry evaluates a single entry from the evaluation dataset
func (te *TrajectoryEvaluator) evaluateEntry(entry EvaluationEntry) (EvaluationResult, *FailureInfo, error) {
	// Get the expected tool use
	expectedToolUses, err := entry.GetExpectedToolUse()
	if err != nil {
		return EvaluationResult{}, nil, err
	}

	// Get the actual tool use
	actualToolUses, err := entry.GetActualToolUse()
	if err != nil {
		return EvaluationResult{}, nil, err
	}

	// Remove tool outputs from expected tool uses
	expectedToolUsesWithoutOutputs := te.removeToolOutputs(expectedToolUses)

	// Compare tool trajectories
	toolUseAccuracy := 0.0
	if te.areToolsEqual(actualToolUses, expectedToolUsesWithoutOutputs) {
		toolUseAccuracy = 1.0
	}

	result := EvaluationResult{
		Query:           entry.GetQuery(),
		Response:        entry.GetResponse(),
		ActualToolUse:   actualToolUses,
		ExpectedToolUse: expectedToolUsesWithoutOutputs,
		ToolUseAccuracy: toolUseAccuracy,
	}

	var failure *FailureInfo
	if toolUseAccuracy != 1.0 {
		failure = &FailureInfo{
			Query:    entry.GetQuery(),
			Actual:   actualToolUses,
			Expected: expectedToolUsesWithoutOutputs,
		}
	}

	return result, failure, nil
}

// removeToolOutputs removes the MockToolOutput field from a list of tool uses
func (te *TrajectoryEvaluator) removeToolOutputs(toolUses []ToolUse) []ToolUse {
	result := make([]ToolUse, len(toolUses))

	for i, toolUse := range toolUses {
		result[i] = ToolUse{
			ToolName:  toolUse.ToolName,
			ToolInput: toolUse.ToolInput,
		}
	}

	return result
}

// areToolsEqual compares two lists of tool uses for equality
func (te *TrajectoryEvaluator) areToolsEqual(listA, listB []ToolUse) bool {
	if len(listA) != len(listB) {
		return false
	}

	for i := range listA {
		if listA[i].ToolName != listB[i].ToolName {
			return false
		}

		// Compare tool inputs
		if !reflect.DeepEqual(listA[i].ToolInput, listB[i].ToolInput) {
			return false
		}
	}

	return true
}

// reportFailures prints details about failed tool evaluations
func (te *TrajectoryEvaluator) reportFailures(failures []FailureInfo) {
	if len(failures) == 0 {
		return
	}

	fmt.Println("Failures:")
	for _, failure := range failures {
		actualBytes, _ := json.MarshalIndent(failure.Actual, "", "  ")
		expectedBytes, _ := json.MarshalIndent(failure.Expected, "", "  ")

		fmt.Printf(`{
  "turn": %d,
  "query": "%s",
  "actual": %s,
  "expected_tool_use": %s
}
`, failure.Turn, failure.Query, string(actualBytes), string(expectedBytes))
	}
}

// printResults prints detailed results of the evaluation
func (te *TrajectoryEvaluator) printResults(results []EvaluationResult) {
	fmt.Println("\nTrajectory Evaluation Results:")
	fmt.Println("--------------------------------")

	for i, result := range results {
		fmt.Printf("Entry %d:\n", i+1)
		fmt.Printf("  Query: %s\n", result.Query)
		fmt.Printf("  Response: %s\n", result.Response)
		fmt.Printf("  Tool Use Accuracy: %.2f\n", result.ToolUseAccuracy)

		fmt.Printf("  Expected Tool Use:\n")
		for j, toolUse := range result.ExpectedToolUse {
			toolInputBytes, _ := json.MarshalIndent(toolUse.ToolInput, "    ", "  ")
			fmt.Printf("    %d. %s: %s\n", j+1, toolUse.ToolName, string(toolInputBytes))
		}

		fmt.Printf("  Actual Tool Use:\n")
		for j, toolUse := range result.ActualToolUse {
			toolInputBytes, _ := json.MarshalIndent(toolUse.ToolInput, "    ", "  ")
			fmt.Printf("    %d. %s: %s\n", j+1, toolUse.ToolName, string(toolInputBytes))
		}

		fmt.Println("--------------------------------")
	}

	// Calculate and print mean accuracy
	var totalAccuracy float64
	for _, result := range results {
		totalAccuracy += result.ToolUseAccuracy
	}

	if len(results) > 0 {
		fmt.Printf("Mean Tool Use Accuracy: %.2f\n", totalAccuracy/float64(len(results)))
	}
}
