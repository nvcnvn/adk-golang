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

// Package tools provides a collection of tools that can be used by agents.
// Tools represent capabilities that can be provided to agents, such as
// searching the web, executing code, or interacting with other agents.
package tools

// This file exports all the tools defined in the package to make them
// easily accessible to users of the package.

import (
	"fmt"
)

// CreateVertexAISearchWithDataStore is a convenience function to create a Vertex AI Search
// tool with a data store ID. It returns an error if creation fails.
func CreateVertexAISearchWithDataStore(dataStoreID string) (*VertexAISearchTool, error) {
	if dataStoreID == "" {
		return nil, fmt.Errorf("dataStoreID cannot be empty")
	}
	return VertexAISearchWithDataStore(dataStoreID)
}

// CreateVertexAISearchWithEngine is a convenience function to create a Vertex AI Search
// tool with a search engine ID. It returns an error if creation fails.
func CreateVertexAISearchWithEngine(searchEngineID string) (*VertexAISearchTool, error) {
	if searchEngineID == "" {
		return nil, fmt.Errorf("searchEngineID cannot be empty")
	}
	return VertexAISearchWithEngine(searchEngineID)
}
