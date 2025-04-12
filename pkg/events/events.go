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

// Package events provides functionality for event handling.
package events

import (
	"sync"
	"time"
)

// EventType defines the type of event.
type EventType string

// Standard event types.
const (
	// Agent events
	AgentStarted         EventType = "agent.started"
	AgentStopped         EventType = "agent.stopped"
	AgentInputReceived   EventType = "agent.input.received"
	AgentOutputGenerated EventType = "agent.output.generated"

	// Tool events
	ToolCalled         EventType = "tool.called"
	ToolResultReceived EventType = "tool.result.received"
	ToolError          EventType = "tool.error"

	// Model events
	ModelCalled           EventType = "model.called"
	ModelResponseReceived EventType = "model.response.received"
	ModelError            EventType = "model.error"
)

// Event represents an event in the system.
type Event struct {
	// Type is the type of the event.
	Type EventType

	// Timestamp is when the event occurred.
	Timestamp time.Time

	// Payload is the data associated with the event.
	Payload interface{}
}

// Handler is a function that handles events.
type Handler func(event Event)

// EventBus manages event subscriptions and dispatches events.
type EventBus struct {
	subscribers map[EventType][]Handler
	mu          sync.RWMutex
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]Handler),
	}
}

// Subscribe registers a handler for the given event type.
func (b *EventBus) Subscribe(eventType EventType, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

// SubscribeAll registers a handler for all event types.
func (b *EventBus) SubscribeAll(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers["*"] = append(b.subscribers["*"], handler)
}

// Publish dispatches an event to all subscribers.
func (b *EventBus) Publish(eventType EventType, payload interface{}) {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Payload:   payload,
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Notify type-specific subscribers
	if handlers, ok := b.subscribers[eventType]; ok {
		for _, handler := range handlers {
			go handler(event)
		}
	}

	// Notify subscribers to all events
	if handlers, ok := b.subscribers["*"]; ok {
		for _, handler := range handlers {
			go handler(event)
		}
	}
}

// Default global event bus
var (
	defaultBus  *EventBus
	defaultOnce sync.Once
)

// DefaultBus returns the default global event bus.
func DefaultBus() *EventBus {
	defaultOnce.Do(func() {
		defaultBus = NewEventBus()
	})
	return defaultBus
}

// Subscribe is a convenience function that subscribes to the default bus.
func Subscribe(eventType EventType, handler Handler) {
	DefaultBus().Subscribe(eventType, handler)
}

// SubscribeAll is a convenience function that subscribes to all events on the default bus.
func SubscribeAll(handler Handler) {
	DefaultBus().SubscribeAll(handler)
}

// Publish is a convenience function that publishes to the default bus.
func Publish(eventType EventType, payload interface{}) {
	DefaultBus().Publish(eventType, payload)
}
