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

package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/models"
	"github.com/nvcnvn/adk-golang/pkg/telemetry"
)

// RemoteAgent represents an agent that runs remotely via HTTP API calls.
type RemoteAgent struct {
	name        string
	url         string
	description string
	httpClient  *http.Client
	parentAgent BaseAgent
}

// NewRemoteAgent creates a new remote agent
func NewRemoteAgent(name, url, description string) *RemoteAgent {
	return &RemoteAgent{
		name:        name,
		url:         url,
		description: description,
		httpClient:  &http.Client{Timeout: 120 * time.Second},
	}
}

// Name returns the name of the agent
func (a *RemoteAgent) Name() string {
	return a.name
}

// Description returns the description of the agent
func (a *RemoteAgent) Description() string {
	if a.description == "" {
		return "Remote agent communicating via HTTP"
	}
	return a.description
}

// Run executes the agent with the given invocation context
func (a *RemoteAgent) Run(ctx context.Context, invocationContext *InvocationContext) (<-chan *events.Event, error) {
	eventCh := make(chan *events.Event)

	ctx, span := telemetry.StartSpan(ctx, "RemoteAgent.Run")
	defer span.End()

	span.SetAttribute("agent.name", a.name)
	span.SetAttribute("agent.url", a.url)

	data := map[string]interface{}{
		"invocation_id": invocationContext.InvocationID,
		"context": map[string]interface{}{
			"invocation_id": invocationContext.InvocationID,
			"agent_name":    a.name,
		},
	}

	go func() {
		defer close(eventCh)

		jsonData, err := json.Marshal(data)
		if err != nil {
			span.SetAttribute("error", err.Error())
			eventCh <- &events.Event{
				InvocationID: invocationContext.InvocationID,
				Author:       a.name,
				Content: &events.Content{
					Parts: []*models.Part{
						{Text: fmt.Sprintf("Error preparing request: %v", err)},
					},
				},
			}
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url, bytes.NewBuffer(jsonData))
		if err != nil {
			span.SetAttribute("error", err.Error())
			eventCh <- &events.Event{
				InvocationID: invocationContext.InvocationID,
				Author:       a.name,
				Content: &events.Content{
					Parts: []*models.Part{
						{Text: fmt.Sprintf("Error creating request: %v", err)},
					},
				},
			}
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.httpClient.Do(req)
		if err != nil {
			span.SetAttribute("error", err.Error())
			eventCh <- &events.Event{
				InvocationID: invocationContext.InvocationID,
				Author:       a.name,
				Content: &events.Content{
					Parts: []*models.Part{
						{Text: fmt.Sprintf("Error communicating with remote agent: %v", err)},
					},
				},
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errMsg := fmt.Sprintf("Remote agent returned error status: %d", resp.StatusCode)
			span.SetAttribute("error", errMsg)
			eventCh <- &events.Event{
				InvocationID: invocationContext.InvocationID,
				Author:       a.name,
				Content: &events.Content{
					Parts: []*models.Part{
						{Text: errMsg},
					},
				},
			}
			return
		}

		var responseEvents []*events.Event
		if err := json.NewDecoder(resp.Body).Decode(&responseEvents); err != nil {
			span.SetAttribute("error", err.Error())
			eventCh <- &events.Event{
				InvocationID: invocationContext.InvocationID,
				Author:       a.name,
				Content: &events.Content{
					Parts: []*models.Part{
						{Text: fmt.Sprintf("Error parsing response from remote agent: %v", err)},
					},
				},
			}
			return
		}

		for _, event := range responseEvents {
			event.Author = a.name
			eventCh <- event
		}
	}()

	return eventCh, nil
}

// RunLive executes the agent in live mode with the given invocation context
func (a *RemoteAgent) RunLive(ctx context.Context, invocationContext *InvocationContext) (<-chan *events.Event, error) {
	return a.Run(ctx, invocationContext)
}

// RootAgent returns the root agent in the agent tree
func (a *RemoteAgent) RootAgent() BaseAgent {
	if a.parentAgent == nil {
		return a
	}
	return a.parentAgent.RootAgent()
}

// FindAgent finds an agent by name in the agent tree
func (a *RemoteAgent) FindAgent(name string) BaseAgent {
	if a.name == name {
		return a
	}
	return nil
}

// SetParentAgent sets the parent agent
func (a *RemoteAgent) SetParentAgent(parent BaseAgent) {
	a.parentAgent = parent
}
