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

package events

import (
	"sync"
)

// EventType represents the type of an event
type EventType string

// Predefined event types
const (
	ToolCalled         EventType = "tool_called"
	ToolError          EventType = "tool_error"
	ToolResultReceived EventType = "tool_result_received"
)

// EventHandler is a function that handles an event
type EventHandler func(eventType EventType, data map[string]interface{})

// eventBus is the global event bus
var eventBus struct {
	handlers map[EventType][]EventHandler
	mu       sync.RWMutex
}

// init initializes the event bus
func init() {
	eventBus.handlers = make(map[EventType][]EventHandler)
}

// Subscribe registers a handler for an event type
func Subscribe(eventType EventType, handler EventHandler) {
	eventBus.mu.Lock()
	defer eventBus.mu.Unlock()

	eventBus.handlers[eventType] = append(eventBus.handlers[eventType], handler)
}

// Unsubscribe removes a handler for an event type
func Unsubscribe(eventType EventType, handler EventHandler) {
	eventBus.mu.Lock()
	defer eventBus.mu.Unlock()

	handlers, ok := eventBus.handlers[eventType]
	if !ok {
		return
	}

	var newHandlers []EventHandler
	for _, h := range handlers {
		// Compare function pointer addresses
		if &h != &handler {
			newHandlers = append(newHandlers, h)
		}
	}
	eventBus.handlers[eventType] = newHandlers
}

// Publish sends an event to all subscribers
func Publish(eventType EventType, data map[string]interface{}) {
	eventBus.mu.RLock()
	defer eventBus.mu.RUnlock()

	handlers, ok := eventBus.handlers[eventType]
	if !ok {
		return
	}

	for _, handler := range handlers {
		handler(eventType, data)
	}
}
