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

	"github.com/nvcnvn/adk-golang/pkg/agents"
	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/models"
)

// ContentsProcessor processes LLM request contents before sending to the model
type ContentsProcessor struct{}

// NewContentsProcessor creates a new ContentsProcessor
func NewContentsProcessor() *ContentsProcessor {
	return &ContentsProcessor{}
}

// Run processes the LLM request contents
func (p *ContentsProcessor) Run(ctx context.Context, invocationContext *agents.InvocationContext, llmRequest *models.LlmRequest) (<-chan *events.Event, error) {
	eventCh := make(chan *events.Event)

	go func() {
		defer close(eventCh)

		// Initialize content if not present
		if llmRequest.Contents == nil {
			llmRequest.Contents = &models.Content{
				Parts: make([]*models.Part, 0),
			}
		}

		// Add invocation event to contents if present
		if invocationContext.InvocationEvent != nil && invocationContext.InvocationEvent.Content != nil {
			llmRequest.Contents.Parts = append(llmRequest.Contents.Parts, &models.Part{
				Text: invocationContext.InvocationEvent.Content.GetText(),
				Role: "user",
			})
		}

		// Iterate through events and build history
		history := buildHistoryFromEvents(invocationContext.Events)

		// Add history to the request contents
		for _, part := range history.Parts {
			llmRequest.Contents.Parts = append(llmRequest.Contents.Parts, part)
		}
	}()

	return eventCh, nil
}

// buildHistoryFromEvents constructs a content history from events
func buildHistoryFromEvents(events []*events.Event) *models.Content {
	content := &models.Content{
		Parts: make([]*models.Part, 0),
	}

	for _, event := range events {
		if event.Content == nil {
			continue
		}

		for _, part := range event.Content.Parts {
			// Determine the role based on the event author
			role := "assistant"
			if event.Author == "user" {
				role = "user"
			}

			// Create a new part with the appropriate role
			newPart := &models.Part{
				Role: role,
			}

			// Copy content based on type
			if part.Text != "" {
				newPart.Text = part.Text
			} else if part.FunctionCall != nil {
				newPart.FunctionCall = part.FunctionCall
			} else if part.FunctionResponse != nil {
				newPart.FunctionResponse = part.FunctionResponse
			}

			content.Parts = append(content.Parts, newPart)
		}
	}

	return content
}
