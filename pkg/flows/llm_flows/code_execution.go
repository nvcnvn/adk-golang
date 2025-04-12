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

package llm_flows

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/nvcnvn/adk-golang/pkg/agents"
	"github.com/nvcnvn/adk-golang/pkg/code_executors"
	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/models"
)

// CodeExecutionProcessor handles code execution in LLM flows
type CodeExecutionProcessor struct {
	CodeExecutor code_executors.CodeExecutor
}

// NewCodeExecutionProcessor creates a new CodeExecutionProcessor
func NewCodeExecutionProcessor(codeExecutor code_executors.CodeExecutor) *CodeExecutionProcessor {
	return &CodeExecutionProcessor{
		CodeExecutor: codeExecutor,
	}
}

// Run processes LLM responses looking for code to execute
func (p *CodeExecutionProcessor) Run(ctx context.Context, invocationContext *agents.InvocationContext, llmResponse *models.LlmResponse) (<-chan *events.Event, error) {
	eventCh := make(chan *events.Event)

	go func() {
		defer close(eventCh)

		if llmResponse.Partial {
			return
		}

		// Check if there is content to process
		if llmResponse.Content == nil || len(llmResponse.Content.Parts) == 0 {
			return
		}

		// Look for code blocks in the content
		codeBlocks := extractCodeBlocks(llmResponse.Content)
		if len(codeBlocks) == 0 {
			return
		}

		// Execute each code block
		for _, codeBlock := range codeBlocks {
			// Execute using the existing CodeExecutor interface
			result, err := p.CodeExecutor.Execute(ctx, codeBlock.Code, []code_executors.File{})

			if err != nil {
				log.Printf("Error executing code: %v", err)
				continue
			}

			// Create an event with the execution result
			event := events.NewEvent()
			event.InvocationID = invocationContext.InvocationID
			event.Author = invocationContext.Agent.Name()
			event.Content = &models.Content{
				Parts: []*models.Part{
					{
						Text: formatExecutionResult(result),
					},
				},
			}

			eventCh <- event
		}
	}()

	return eventCh, nil
}

// CodeBlock represents a block of code to be executed
type CodeBlock struct {
	Code     string
	Language string
}

// extractCodeBlocks extracts code blocks from LLM content
func extractCodeBlocks(content *models.Content) []*CodeBlock {
	codeBlocks := make([]*CodeBlock, 0)

	for _, part := range content.Parts {
		if part.Text == "" {
			continue
		}

		// Simple code block extraction logic
		// This can be enhanced to match more complex patterns
		lines := strings.Split(part.Text, "\n")
		var currentBlock *CodeBlock

		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				if currentBlock == nil {
					// Start of a code block
					language := strings.TrimPrefix(line, "```")
					currentBlock = &CodeBlock{
						Language: language,
					}
				} else {
					// End of a code block
					codeBlocks = append(codeBlocks, currentBlock)
					currentBlock = nil
				}
			} else if currentBlock != nil {
				// Inside a code block
				currentBlock.Code += line + "\n"
			}
		}
	}

	return codeBlocks
}

// formatExecutionResult formats code execution results
func formatExecutionResult(result *code_executors.ExecutionResult) string {
	var sb strings.Builder

	sb.WriteString("```\n")

	// Add stdout if present
	if result.Stdout != "" {
		sb.WriteString(result.Stdout)
	}

	// Add stderr if present (as an error)
	if result.Stderr != "" {
		if result.Stdout != "" {
			sb.WriteString("\n")
		}
		sb.WriteString("Error: ")
		sb.WriteString(result.Stderr)
	}

	// Add any output files
	if len(result.OutputFiles) > 0 {
		if result.Stdout != "" || result.Stderr != "" {
			sb.WriteString("\n\n")
		}
		sb.WriteString("Output files:\n")

		for _, file := range result.OutputFiles {
			sb.WriteString(fmt.Sprintf("- %s\n", file.Name))
		}
	}

	sb.WriteString("\n```\n")

	return sb.String()
}
