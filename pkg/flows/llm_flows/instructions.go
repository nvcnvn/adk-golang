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

// InstructionsProcessor adds system instructions to LLM requests
type InstructionsProcessor struct{}

// NewInstructionsProcessor creates a new InstructionsProcessor
func NewInstructionsProcessor() *InstructionsProcessor {
	return &InstructionsProcessor{}
}

// Run processes the LLM request by adding system instructions
func (p *InstructionsProcessor) Run(ctx context.Context, invocationContext *agents.InvocationContext, llmRequest *models.LlmRequest) (<-chan *events.Event, error) {
	eventCh := make(chan *events.Event)

	go func() {
		defer close(eventCh)

		llmAgent, ok := invocationContext.Agent.(*agents.LlmAgent)
		if !ok {
			return
		}

		// Get system instructions from the agent
		instructions := llmAgent.SystemInstructions
		if instructions == "" {
			return
		}

		// Initialize content if not present
		if llmRequest.Contents == nil {
			llmRequest.Contents = &models.Content{
				Parts: make([]*models.Part, 0),
			}
		}

		// Create a system part with the instructions
		systemPart := &models.Part{
			Text: instructions,
			Role: "system",
		}

		// Add the system part at the beginning of the content parts
		newParts := []*models.Part{systemPart}
		newParts = append(newParts, llmRequest.Contents.Parts...)
		llmRequest.Contents.Parts = newParts
	}()

	return eventCh, nil
}
