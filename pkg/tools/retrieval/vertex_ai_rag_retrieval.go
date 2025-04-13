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

package retrieval

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// RagResource represents a Vertex AI RAG resource.
type RagResource struct {
	Name string
}

// VertexRagStore contains configuration for Vertex AI RAG retrieval.
type VertexRagStore struct {
	RagCorpora              []string
	RagResources            []RagResource
	SimilarityTopK          int
	VectorDistanceThreshold float64
}

// VertexAiRagRetrieval implements a retrieval tool that uses Vertex AI RAG to retrieve data.
type VertexAiRagRetrieval struct {
	*BaseRetrievalTool
	vertexRagStore VertexRagStore
}

// NewVertexAiRagRetrieval creates a new VertexAiRagRetrieval tool.
func NewVertexAiRagRetrieval(name, description string, opts ...VertexAiRagOption) *VertexAiRagRetrieval {
	store := VertexRagStore{}

	for _, opt := range opts {
		opt(&store)
	}

	return &VertexAiRagRetrieval{
		BaseRetrievalTool: NewBaseRetrievalTool(name, description),
		vertexRagStore:    store,
	}
}

// VertexAiRagOption is a function that configures a VertexRagStore.
type VertexAiRagOption func(*VertexRagStore)

// WithRagCorpora sets the RAG corpora.
func WithRagCorpora(corpora []string) VertexAiRagOption {
	return func(s *VertexRagStore) {
		s.RagCorpora = corpora
	}
}

// WithRagResources sets the RAG resources.
func WithRagResources(resources []RagResource) VertexAiRagOption {
	return func(s *VertexRagStore) {
		s.RagResources = resources
	}
}

// WithSimilarityTopK sets the similarity top k value.
func WithSimilarityTopK(k int) VertexAiRagOption {
	return func(s *VertexRagStore) {
		s.SimilarityTopK = k
	}
}

// WithVectorDistanceThreshold sets the vector distance threshold value.
func WithVectorDistanceThreshold(threshold float64) VertexAiRagOption {
	return func(s *VertexRagStore) {
		s.VectorDistanceThreshold = threshold
	}
}

// Execute runs the retrieval using Vertex AI RAG.
func (v *VertexAiRagRetrieval) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	query, ok := input["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	// In a real implementation, we would create a Discovery Engine client here
	// For now, we'll log a message about what would happen

	log.Printf("Would query Vertex AI RAG with query: %s", query)

	// Execute RAG query for each corpus or resource
	var results []string

	if len(v.vertexRagStore.RagCorpora) > 0 {
		for _, corpus := range v.vertexRagStore.RagCorpora {
			contents, err := v.queryCorpus(ctx, corpus, query)
			if err != nil {
				log.Printf("Error querying corpus %s: %v", corpus, err)
				continue
			}
			results = append(results, contents...)
		}
	}

	if len(v.vertexRagStore.RagResources) > 0 {
		for _, resource := range v.vertexRagStore.RagResources {
			contents, err := v.queryResource(ctx, resource.Name, query)
			if err != nil {
				log.Printf("Error querying resource %s: %v", resource.Name, err)
				continue
			}
			results = append(results, contents...)
		}
	}

	if len(results) == 0 {
		return map[string]interface{}{
			"result": fmt.Sprintf("No matching result found with the config: %+v", v.vertexRagStore),
		}, nil
	}

	// Return all retrieved contents
	return map[string]interface{}{
		"result": results,
	}, nil
}

// queryCorpus queries a specific corpus with the given query.
func (v *VertexAiRagRetrieval) queryCorpus(ctx context.Context, corpus string, query string) ([]string, error) {
	// This is a placeholder for the actual implementation
	log.Printf("Querying corpus: %s with query: %s", corpus, query)

	// In real implementation, we would call the Vertex AI Discovery Engine API
	// For now, we return a placeholder result
	return []string{fmt.Sprintf("Sample result from corpus %s for query %s", corpus, query)}, nil
}

// queryResource queries a specific resource with the given query.
func (v *VertexAiRagRetrieval) queryResource(ctx context.Context, resourceName string, query string) ([]string, error) {
	// This is a placeholder for the actual implementation
	log.Printf("Querying resource: %s with query: %s", resourceName, query)

	// In real implementation, we would call the Vertex AI Discovery Engine API
	// For now, we return a placeholder result
	return []string{fmt.Sprintf("Sample result from resource %s for query %s", resourceName, query)}, nil
}

// ProcessLlmRequest processes an LLM request by configuring LLM with RAG capabilities.
// This is intended to be used with LLM integrations in your Golang application.
func (v *VertexAiRagRetrieval) ProcessLlmRequest(ctx context.Context, llmRequest interface{}) error {
	// This is a placeholder for integrating with LLM requests

	// Example: convert llmRequest to JSON and add RAG configuration
	jsonBytes, err := json.Marshal(map[string]interface{}{
		"tools": []map[string]interface{}{
			{
				"retrieval": map[string]interface{}{
					"vertex_rag_store": v.vertexRagStore,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to serialize LLM request: %w", err)
	}

	log.Printf("LLM request with RAG configuration: %s", string(jsonBytes))

	return nil
}
