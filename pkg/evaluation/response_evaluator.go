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
	"fmt"
	"strings"
)

// ResponseEvaluator evaluates agent responses against references
type ResponseEvaluator struct{}

// ResponseMetrics contains metrics from response evaluation
type ResponseMetrics map[string]float64

// NewResponseEvaluator creates a new response evaluator
func NewResponseEvaluator() *ResponseEvaluator {
	return &ResponseEvaluator{}
}

// Evaluate evaluates the responses in the dataset against references
// Currently supports calculating ROUGE-1 scores for text similarity
func (re *ResponseEvaluator) Evaluate(
	dataset EvaluationDataset,
	criteria map[string]float64,
	printDetailedResults bool,
) (ResponseMetrics, error) {
	if len(dataset) == 0 {
		return nil, fmt.Errorf("the evaluation dataset is empty")
	}

	metrics := ResponseMetrics{}

	// Check which metrics we need to calculate
	needsCoherence := false
	needsRouge := false

	for key := range criteria {
		if key == ResponseEvaluationScoreKey {
			needsCoherence = true
		} else if key == ResponseMatchScoreKey {
			needsRouge = true
		}
	}

	// Collect all responses and references for metrics calculation
	var allRougeScores []float64
	var allCoherenceScores []float64

	// Process each entry in the dataset and calculate metrics
	for _, conversation := range dataset {
		for _, entry := range conversation {
			response := entry.GetResponse()
			reference := entry.GetReference()

			// Calculate Rouge-1 score if needed
			if needsRouge && reference != "" {
				rougeScore := re.calculateRouge1Score(response, reference)
				allRougeScores = append(allRougeScores, rougeScore)
			}

			// In a real implementation, coherence would be calculated using an LLM
			// Here we simplify with a placeholder implementation
			if needsCoherence {
				// For now, we use a simplified coherence calculation
				// In Python, this uses VertexAI's evaluation capabilities
				coherenceScore := re.calculateSimpleCoherence(response)
				allCoherenceScores = append(allCoherenceScores, coherenceScore)
			}
		}
	}

	// Calculate mean scores
	if len(allRougeScores) > 0 {
		metrics["rouge_1/mean"] = re.calculateMean(allRougeScores)
	}

	if len(allCoherenceScores) > 0 {
		metrics["coherence/mean"] = re.calculateMean(allCoherenceScores)
	}

	// Print detailed results if requested
	if printDetailedResults {
		re.printResults(metrics)
	}

	return metrics, nil
}

// calculateRouge1Score calculates a simplified ROUGE-1 score between response and reference
// This is a simplified version that counts word overlap divided by total unique words
func (re *ResponseEvaluator) calculateRouge1Score(response, reference string) float64 {
	if response == "" || reference == "" {
		return 0.0
	}

	// Tokenize into words
	responseWords := re.tokenize(response)
	referenceWords := re.tokenize(reference)

	// Count matching words
	matches := 0
	for _, rWord := range responseWords {
		for _, refWord := range referenceWords {
			if rWord == refWord {
				matches++
				break
			}
		}
	}

	// Calculate score: matches / reference word count
	if len(referenceWords) == 0 {
		return 0.0
	}

	return float64(matches) / float64(len(referenceWords))
}

// calculateSimpleCoherence calculates a simplified coherence score
// In a production version, this would use an LLM to evaluate coherence
func (re *ResponseEvaluator) calculateSimpleCoherence(response string) float64 {
	// This is a placeholder implementation
	// In a real implementation, this would use an LLM to evaluate coherence
	// based on various factors like grammar, relevance, etc.

	// For now, we return a value between 0 and 5 based on response length
	// Just as a placeholder until proper LLM evaluation is implemented
	words := re.tokenize(response)
	wordCount := len(words)

	if wordCount == 0 {
		return 0.0
	} else if wordCount < 5 {
		return 1.0
	} else if wordCount < 20 {
		return 3.0
	} else {
		return 4.5
	}
}

// tokenize splits text into words, removing punctuation and converting to lowercase
func (re *ResponseEvaluator) tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Replace punctuation with spaces
	replacer := strings.NewReplacer(
		".", " ", ",", " ", "!", " ", "?", " ", ";", " ", ":", " ",
		"(", " ", ")", " ", "[", " ", "]", " ", "{", " ", "}", " ",
		"\"", " ", "'", " ", "\n", " ", "\t", " ",
	)
	text = replacer.Replace(text)

	// Split by spaces and filter empty strings
	tokens := strings.Fields(text)
	return tokens
}

// calculateMean calculates the mean of a slice of float64 values
func (re *ResponseEvaluator) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	var sum float64
	for _, value := range values {
		sum += value
	}

	return sum / float64(len(values))
}

// printResults prints the evaluation metrics
func (re *ResponseEvaluator) printResults(metrics ResponseMetrics) {
	fmt.Println("\nResponse Evaluation Results:")
	fmt.Println("--------------------------")
	for metric, value := range metrics {
		fmt.Printf("%-20s: %.3f\n", metric, value)
	}
	fmt.Println("--------------------------")
}
