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
	"fmt"
)

// Retriever defines the interface for a component that can retrieve information based on a query.
// This is a simplified interface resembling the llama-index BaseRetriever in Python.
type Retriever interface {
	// Retrieve returns relevant documents for the given query
	Retrieve(ctx context.Context, query string) ([]Document, error)
}

// Document represents a document returned from a retrieval operation
type Document struct {
	Text string
}

// LlamaIndexRetrieval implements a retrieval tool using a Retriever interface.
// This is a Golang equivalent to the Python LlamaIndexRetrieval class.
type LlamaIndexRetrieval struct {
	*BaseRetrievalTool
	retriever Retriever
}

// NewLlamaIndexRetrieval creates a new LlamaIndexRetrieval tool.
func NewLlamaIndexRetrieval(name, description string, retriever Retriever) *LlamaIndexRetrieval {
	return &LlamaIndexRetrieval{
		BaseRetrievalTool: NewBaseRetrievalTool(name, description),
		retriever:         retriever,
	}
}

// Execute runs the retrieval using the provided retriever.
func (l *LlamaIndexRetrieval) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	query, ok := input["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	docs, err := l.retriever.Retrieve(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error during retrieval: %w", err)
	}

	if len(docs) == 0 {
		return map[string]interface{}{
			"result": "No relevant information found.",
		}, nil
	}

	// Return the text of the first document, similar to the Python implementation
	return map[string]interface{}{
		"result": docs[0].Text,
	}, nil
}
