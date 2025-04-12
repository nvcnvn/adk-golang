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
	"fmt"

	"github.com/nvcnvn/adk-golang/pkg/events"
	"github.com/nvcnvn/adk-golang/pkg/types"
)

// LiveRequest represents a real-time request from a client
type LiveRequest struct {
	// Content is the text content of the request
	Content *events.Content `json:"content,omitempty"`

	// Blob contains binary data (e.g., audio)
	Blob []byte `json:"blob,omitempty"`

	// Close indicates if the connection should be closed
	Close bool `json:"close,omitempty"`
}

// LiveRequestQueue manages a queue of live requests
type LiveRequestQueue struct {
	queue  chan *LiveRequest
	closed bool
}

// NewLiveRequestQueue creates a new LiveRequestQueue
func NewLiveRequestQueue() *LiveRequestQueue {
	return &LiveRequestQueue{
		queue: make(chan *LiveRequest, 100), // Buffer size can be adjusted
	}
}

// Send adds a request to the queue
func (q *LiveRequestQueue) Send(request *LiveRequest) error {
	if q.closed {
		return fmt.Errorf("queue is closed")
	}
	q.queue <- request
	return nil
}

// SendContent creates and sends a content request
func (q *LiveRequestQueue) SendContent(content *events.Content) error {
	return q.Send(&LiveRequest{Content: content})
}

// Get retrieves the next request from the queue
func (q *LiveRequestQueue) Get() (*LiveRequest, error) {
	if q.closed {
		return nil, fmt.Errorf("queue is closed")
	}
	req, ok := <-q.queue
	if !ok {
		return nil, fmt.Errorf("queue is closed")
	}
	return req, nil
}

// Close closes the queue
func (q *LiveRequestQueue) Close() {
	if !q.closed {
		q.closed = true
		close(q.queue)
	}
}

// ActiveStreamingTool represents a tool with an active streaming connection
type ActiveStreamingTool struct {
	// Stream is the stream for the tool
	Stream Stream
}

// Stream defines an interface for streaming data
type Stream interface {
	// Send sends data to the stream
	Send(data interface{}) error

	// Close closes the stream
	Close() error
}

// InvocationContext holds the context for an agent invocation
type InvocationContext struct {
	// Common data shared with other packages
	types.InvocationContextData

	// Agent is the agent being invoked
	Agent BaseAgent `json:"-"`

	// InvocationEvent is the event that triggered this invocation
	InvocationEvent *events.Event `json:"invocationEvent,omitempty"`

	// Events contains all events associated with this invocation
	Events []*events.Event `json:"events,omitempty"`

	// LiveRequestQueue holds the queue for live requests
	LiveRequestQueue *LiveRequestQueue `json:"-"`

	// ActiveStreamingTools holds active streaming tools
	ActiveStreamingTools map[string]*ActiveStreamingTool `json:"-"`
}

// NewInvocationContext creates a new InvocationContext
func NewInvocationContext(invocationID string, agent BaseAgent, runConfig *types.RunConfig) *InvocationContext {
	if runConfig == nil {
		runConfig = &types.RunConfig{
			StreamingMode: types.StreamingModeNone,
			MaxLlmCalls:   10,
		}
	}

	return &InvocationContext{
		InvocationContextData: types.InvocationContextData{
			InvocationID: invocationID,
			RunConfig:    runConfig,
		},
		Agent:                agent,
		Events:               make([]*events.Event, 0),
		ActiveStreamingTools: make(map[string]*ActiveStreamingTool),
	}
}

// GetID returns the invocation ID
func (ctx *InvocationContext) GetID() string {
	return ctx.InvocationID
}

// GetAgentName returns the name of the agent
func (ctx *InvocationContext) GetAgentName() string {
	return ctx.Agent.Name()
}

// IsEndInvocation returns whether the invocation should end
func (ctx *InvocationContext) IsEndInvocation() bool {
	return ctx.EndInvocation
}

// SetEndInvocation sets whether the invocation should end
func (ctx *InvocationContext) SetEndInvocation(end bool) {
	ctx.EndInvocation = end
}

// GetTranscriptionCache returns the transcription cache
func (ctx *InvocationContext) GetTranscriptionCache() interface{} {
	return ctx.TranscriptionCache
}

// CallbackContext represents a context for agent callbacks
type CallbackContext struct {
	// InvocationContext is the parent invocation context
	InvocationContext *InvocationContext `json:"-"`

	// EventActions contains actions associated with the event
	EventActions *events.EventActions `json:"eventActions,omitempty"`
}
