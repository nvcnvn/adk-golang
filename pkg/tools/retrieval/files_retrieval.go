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
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// DirectoryRetriever implements Retriever interface for files in a directory.
// This is a simplified equivalent of SimpleDirectoryReader + VectorStoreIndex in Python.
type DirectoryRetriever struct {
	inputDir string
	// In a real implementation, this would use a vector index for efficient similarity search
}

// NewDirectoryRetriever creates a new DirectoryRetriever.
func NewDirectoryRetriever(inputDir string) (*DirectoryRetriever, error) {
	// Verify directory exists
	info, err := os.Stat(inputDir)
	if err != nil {
		return nil, fmt.Errorf("error accessing input directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("input path is not a directory: %s", inputDir)
	}

	log.Printf("Loading data from %s", inputDir)
	return &DirectoryRetriever{
		inputDir: inputDir,
	}, nil
}

// Retrieve implements the Retriever interface for directory contents.
// This is a simplified implementation - a production version would use embeddings and vector search.
func (d *DirectoryRetriever) Retrieve(ctx context.Context, query string) ([]Document, error) {
	var docs []Document

	err := filepath.WalkDir(d.inputDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		// Skip non-text files (a more robust implementation would check MIME types)
		ext := strings.ToLower(filepath.Ext(path))
		if !isTextFile(ext) {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Warning: Could not read file %s: %v", path, err)
			return nil
		}

		// In a real implementation, we would compute similarity with the query
		// For now, just do basic substring matching
		if strings.Contains(strings.ToLower(string(content)), strings.ToLower(query)) {
			docs = append(docs, Document{
				Text: fmt.Sprintf("File: %s\n\n%s", path, string(content)),
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return docs, nil
}

// isTextFile returns true if the file extension is likely to be a text file.
func isTextFile(ext string) bool {
	textExts := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".py": true, ".js": true,
		".ts": true, ".html": true, ".css": true, ".json": true, ".yaml": true,
		".yml": true, ".xml": true, ".csv": true, ".sh": true, ".bat": true,
	}
	return textExts[ext]
}

// FilesRetrieval implements a retrieval tool for files in a directory.
type FilesRetrieval struct {
	*LlamaIndexRetrieval
}

// NewFilesRetrieval creates a new FilesRetrieval tool.
func NewFilesRetrieval(name, description, inputDir string) (*FilesRetrieval, error) {
	retriever, err := NewDirectoryRetriever(inputDir)
	if err != nil {
		return nil, err
	}

	return &FilesRetrieval{
		LlamaIndexRetrieval: NewLlamaIndexRetrieval(name, description, retriever),
	}, nil
}
