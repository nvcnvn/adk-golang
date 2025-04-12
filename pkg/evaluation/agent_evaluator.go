// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package evaluation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Constants for default evaluation settings
const (
	NumRuns = 2 // Default number of runs
)

// AgentEvaluator evaluates agents using test cases
type AgentEvaluator struct {
	generator           *EvaluationGenerator
	trajectoryEvaluator *TrajectoryEvaluator
	responseEvaluator   *ResponseEvaluator
}

// NewAgentEvaluator creates a new agent evaluator
func NewAgentEvaluator() *AgentEvaluator {
	return &AgentEvaluator{
		generator:           NewEvaluationGenerator(),
		trajectoryEvaluator: NewTrajectoryEvaluator(),
		responseEvaluator:   NewResponseEvaluator(),
	}
}

// Evaluate evaluates an agent given evaluation data
func (ae *AgentEvaluator) Evaluate(
	agent Agent,
	evalDatasetFilePathOrDir string,
	numRuns int,
	agentName string,
	initialSessionFile string,
) error {
	// Find test files
	testFiles, err := ae.findTestFiles(evalDatasetFilePathOrDir)
	if err != nil {
		return err
	}

	// Load initial session state if provided
	initialSession := make(map[string]interface{})
	if initialSessionFile != "" {
		sessionData, err := os.ReadFile(initialSessionFile)
		if err != nil {
			return fmt.Errorf("failed to read initial session file: %v", err)
		}

		var sessionObj map[string]interface{}
		if err := json.Unmarshal(sessionData, &sessionObj); err != nil {
			return fmt.Errorf("failed to parse initial session JSON: %v", err)
		}

		if state, ok := sessionObj["state"].(map[string]interface{}); ok {
			initialSession["state"] = state
		}
	}

	// Process each test file
	for _, testFile := range testFiles {
		// Load dataset
		dataset, err := ae.loadDataset(testFile)
		if err != nil {
			return err
		}

		// Find evaluation criteria for this test file
		criteria, err := ae.findConfigForTestFile(testFile)
		if err != nil {
			return err
		}

		// Validate input
		if err := ae.validateInput(dataset, criteria); err != nil {
			return err
		}

		// Generate responses
		evaluationResponses, err := ae.generator.GenerateResponses(
			dataset,
			agent,
			numRuns,
			agentName,
			initialSession,
		)
		if err != nil {
			return err
		}

		// Evaluate responses if needed
		if ae.responseEvaluationRequired(criteria, dataset) {
			if err := ae.evaluateResponseScores(agent, evaluationResponses, criteria); err != nil {
				return err
			}
		}

		// Evaluate tool trajectory if needed
		if ae.trajectoryEvaluationRequired(criteria, dataset) {
			if err := ae.evaluateToolTrajectory(agent, evaluationResponses, criteria); err != nil {
				return err
			}
		}
	}

	return nil
}

// findTestFiles finds all test files from the provided path
func (ae *AgentEvaluator) findTestFiles(path string) ([]string, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path %s: %v", path, err)
	}

	if fileInfo.IsDir() {
		var testFiles []string
		err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(p, ".test.json") {
				testFiles = append(testFiles, p)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %v", path, err)
		}
		return testFiles, nil
	}

	return []string{path}, nil
}

// loadDataset loads a dataset from a file
func (ae *AgentEvaluator) loadDataset(testFile string) (EvaluationDataset, error) {
	dataset, err := ae.generator.LoadDataset(testFile)
	if err != nil {
		return nil, err
	}

	return dataset, nil
}

// findConfigForTestFile finds the test configuration in the same folder as the test file
func (ae *AgentEvaluator) findConfigForTestFile(testFile string) (map[string]float64, error) {
	testFolder := filepath.Dir(testFile)
	configPath := filepath.Join(testFolder, "test_config.json")

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read test config file: %v", err)
		}

		var configData map[string]interface{}
		if err := json.Unmarshal(data, &configData); err != nil {
			return nil, fmt.Errorf("invalid JSON in test config file: %v", err)
		}

		criteriaI, exists := configData["criteria"]
		if !exists {
			return nil, fmt.Errorf("test_config.json missing 'criteria' field")
		}

		criteriaMap, ok := criteriaI.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("'criteria' must be a dictionary")
		}

		// Convert to map[string]float64
		criteria := make(map[string]float64)
		for key, value := range criteriaMap {
			floatValue, ok := value.(float64)
			if !ok {
				return nil, fmt.Errorf("criteria values must be numbers")
			}
			criteria[key] = floatValue
		}

		return criteria, nil
	}

	// Return default criteria if no config file exists
	return DefaultCriteria, nil
}

// validateInput validates that the evaluation criteria align with the provided dataset
func (ae *AgentEvaluator) validateInput(evalDataset EvaluationDataset, criteria map[string]float64) error {
	if len(evalDataset) == 0 {
		return fmt.Errorf("the evaluation dataset is None or empty")
	}

	// Validate criteria keys
	for key := range criteria {
		isAllowed := false
		for _, allowedKey := range AllowedCriteria {
			if key == allowedKey {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			return fmt.Errorf("invalid criteria key: %s. Expected one of %v", key, AllowedCriteria)
		}
	}

	// Check if dataset is valid
	if len(evalDataset[0]) == 0 {
		return fmt.Errorf("the evaluation dataset is empty")
	}

	firstQuery := evalDataset[0][0]

	// Validate required fields based on criteria
	if _, hasKey := criteria[ToolTrajectoryScoreKey]; hasKey {
		if _, hasQuery := firstQuery[Query]; !hasQuery {
			return fmt.Errorf("samples for %s must include '%s' key", ToolTrajectoryScoreKey, Query)
		}
		if _, hasExpectedToolUse := firstQuery[ExpectedToolUse]; !hasExpectedToolUse {
			return fmt.Errorf("samples for %s must include '%s' key", ToolTrajectoryScoreKey, ExpectedToolUse)
		}
	}

	if _, hasKey := criteria[ResponseEvaluationScoreKey]; hasKey {
		if _, hasQuery := firstQuery[Query]; !hasQuery {
			return fmt.Errorf("samples for %s must include '%s' key", ResponseEvaluationScoreKey, Query)
		}
	}

	if _, hasKey := criteria[ResponseMatchScoreKey]; hasKey {
		if _, hasQuery := firstQuery[Query]; !hasQuery {
			return fmt.Errorf("samples for %s must include '%s' key", ResponseMatchScoreKey, Query)
		}
		if _, hasReference := firstQuery[Reference]; !hasReference {
			return fmt.Errorf("samples for %s must include '%s' key", ResponseMatchScoreKey, Reference)
		}
	}

	return nil
}

// responseEvaluationRequired checks if response evaluation is needed
func (ae *AgentEvaluator) responseEvaluationRequired(criteria map[string]float64, evalDataset EvaluationDataset) bool {
	_, hasReference := evalDataset[0][0][Reference]
	_, hasResponseEval := criteria[ResponseEvaluationScoreKey]
	_, hasResponseMatch := criteria[ResponseMatchScoreKey]

	return hasReference && (hasResponseEval || hasResponseMatch)
}

// trajectoryEvaluationRequired checks if trajectory evaluation is needed
func (ae *AgentEvaluator) trajectoryEvaluationRequired(criteria map[string]float64, evalDataset EvaluationDataset) bool {
	_, hasExpectedToolUse := evalDataset[0][0][ExpectedToolUse]
	_, hasToolTrajectory := criteria[ToolTrajectoryScoreKey]

	return hasExpectedToolUse && hasToolTrajectory
}

// evaluateResponseScores evaluates response scores and asserts they meet criteria
func (ae *AgentEvaluator) evaluateResponseScores(
	agent Agent,
	evaluationResponse EvaluationDataset,
	criteria map[string]float64,
) error {
	metrics, err := ae.responseEvaluator.Evaluate(evaluationResponse, criteria, true)
	if err != nil {
		return err
	}

	// Assert coherence score if needed
	if threshold, ok := criteria[ResponseEvaluationScoreKey]; ok {
		if err := ae.assertScore(
			metrics,
			"coherence/mean",
			threshold,
			"Average response evaluation score",
			agent,
		); err != nil {
			return err
		}
	}

	// Assert response match score if needed
	if threshold, ok := criteria[ResponseMatchScoreKey]; ok {
		if err := ae.assertScore(
			metrics,
			"rouge_1/mean",
			threshold,
			"Average response match score",
			agent,
		); err != nil {
			return err
		}
	}

	return nil
}

// evaluateToolTrajectory evaluates tool trajectory scores and asserts they meet criteria
func (ae *AgentEvaluator) evaluateToolTrajectory(
	agent Agent,
	evaluationResponse EvaluationDataset,
	criteria map[string]float64,
) error {
	score, err := ae.trajectoryEvaluator.Evaluate(evaluationResponse, true)
	if err != nil {
		return err
	}

	return ae.assertScore(
		map[string]float64{ToolTrajectoryScoreKey: score},
		ToolTrajectoryScoreKey,
		criteria[ToolTrajectoryScoreKey],
		"Average tool trajectory evaluation score",
		agent,
	)
}

// assertScore asserts that a metric meets the specified threshold
func (ae *AgentEvaluator) assertScore(
	metrics map[string]float64,
	metricKey string,
	threshold float64,
	description string,
	agent Agent,
) error {
	actualScore, ok := metrics[metricKey]
	if !ok {
		return fmt.Errorf("metric %s not found in evaluation results", metricKey)
	}

	if actualScore < threshold {
		return fmt.Errorf(
			"%s is lower than expected. Expected >= %f, but got %f",
			description, threshold, actualScore,
		)
	}

	return nil
}
