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

package tools

import (
	"context"
	"fmt"
)

// GoogleSearch is a predefined tool for searching the web.
var GoogleSearch = NewTool(
	"google_search",
	"Search the web for real-time information using Google Search.",
	ToolSchema{
		Input: ParameterSchema{
			Type: "object",
			Properties: map[string]ParameterSchema{
				"query": {
					Type:        "string",
					Description: "The search query to execute",
					Required:    true,
				},
				"num_results": {
					Type:        "integer",
					Description: "The number of search results to return",
					Required:    false,
				},
			},
		},
		Output: map[string]ParameterSchema{
			"results": {
				Type:        "array",
				Description: "The search results",
			},
		},
	},
	func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		query, ok := input["query"].(string)
		if !ok {
			return nil, fmt.Errorf("query must be a string")
		}

		numResults := 3 // Default
		if numResultsInput, ok := input["num_results"].(float64); ok {
			numResults = int(numResultsInput)
		}

		// In a real implementation, this would connect to the Google Search API
		// For now, we'll just return a mock response
		mockResults := []map[string]interface{}{
			{
				"title":       "Mock Search Result 1 for: " + query,
				"url":         "https://example.com/1",
				"description": "This is a mock search result for the query: " + query,
			},
		}

		if numResults > 1 {
			mockResults = append(mockResults, map[string]interface{}{
				"title":       "Mock Search Result 2 for: " + query,
				"url":         "https://example.com/2",
				"description": "Another mock search result for the query: " + query,
			})
		}

		if numResults > 2 {
			mockResults = append(mockResults, map[string]interface{}{
				"title":       "Mock Search Result 3 for: " + query,
				"url":         "https://example.com/3",
				"description": "A third mock search result for the query: " + query,
			})
		}

		return map[string]interface{}{
			"results": mockResults,
		}, nil
	},
)
