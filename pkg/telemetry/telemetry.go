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

// Package telemetry provides functionality for collecting metrics and traces.
package telemetry

import (
	"context"
	"log"
	"os"
	"sync"
	"time"
)

// Logger is the default logger for the telemetry package.
var Logger = log.New(os.Stderr, "[ADK] ", log.LstdFlags|log.Lshortfile)

// LogLevel defines the level of logging.
type LogLevel int

const (
	// LevelDebug represents debug level logging.
	LevelDebug LogLevel = iota
	// LevelInfo represents info level logging.
	LevelInfo
	// LevelWarning represents warning level logging.
	LevelWarning
	// LevelError represents error level logging.
	LevelError
)

// CurrentLogLevel controls the current logging level.
var CurrentLogLevel = LevelInfo

// Debug logs a message at debug level.
func Debug(format string, v ...interface{}) {
	if CurrentLogLevel <= LevelDebug {
		Logger.Printf("[DEBUG] "+format, v...)
	}
}

// Info logs a message at info level.
func Info(format string, v ...interface{}) {
	if CurrentLogLevel <= LevelInfo {
		Logger.Printf("[INFO] "+format, v...)
	}
}

// Warning logs a message at warning level.
func Warning(format string, v ...interface{}) {
	if CurrentLogLevel <= LevelWarning {
		Logger.Printf("[WARNING] "+format, v...)
	}
}

// Error logs a message at error level.
func Error(format string, v ...interface{}) {
	if CurrentLogLevel <= LevelError {
		Logger.Printf("[ERROR] "+format, v...)
	}
}

// SetLogLevel sets the current logging level.
func SetLogLevel(level LogLevel) {
	CurrentLogLevel = level
}

// Span represents a unit of work or operation.
type Span interface {
	// End completes the span.
	End()
	// AddEvent adds an event to the span.
	AddEvent(name string, attributes map[string]string)
	// SetAttribute sets an attribute on the span.
	SetAttribute(key, value string)
}

// Tracer creates new spans.
type Tracer interface {
	// Start starts a new span.
	Start(ctx context.Context, name string) (context.Context, Span)
}

// noopSpan is a span that does nothing.
type noopSpan struct{}

func (s *noopSpan) End()                                               {}
func (s *noopSpan) AddEvent(name string, attributes map[string]string) {}
func (s *noopSpan) SetAttribute(key, value string)                     {}

// noopTracer is a tracer that does nothing.
type noopTracer struct{}

func (t *noopTracer) Start(ctx context.Context, name string) (context.Context, Span) {
	return ctx, &noopSpan{}
}

// SimpleSpan is a simple implementation of Span that records events and attributes.
type SimpleSpan struct {
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Events     []SpanEvent
	Attributes map[string]string
	mu         sync.Mutex
}

// SpanEvent represents an event that occurred during a span.
type SpanEvent struct {
	Name       string
	Time       time.Time
	Attributes map[string]string
}

// NewSimpleSpan creates a new SimpleSpan with the given name.
func NewSimpleSpan(name string) *SimpleSpan {
	return &SimpleSpan{
		Name:       name,
		StartTime:  time.Now(),
		Events:     []SpanEvent{},
		Attributes: make(map[string]string),
	}
}

// End completes the span.
func (s *SimpleSpan) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EndTime = time.Now()
}

// AddEvent adds an event to the span.
func (s *SimpleSpan) AddEvent(name string, attributes map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, SpanEvent{
		Name:       name,
		Time:       time.Now(),
		Attributes: attributes,
	})
}

// SetAttribute sets an attribute on the span.
func (s *SimpleSpan) SetAttribute(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Attributes[key] = value
}

// SimpleTracer is a simple implementation of Tracer that records spans.
type SimpleTracer struct {
	spans []*SimpleSpan
	mu    sync.Mutex
}

// NewSimpleTracer creates a new SimpleTracer.
func NewSimpleTracer() *SimpleTracer {
	return &SimpleTracer{
		spans: []*SimpleSpan{},
	}
}

// Start starts a new span.
func (t *SimpleTracer) Start(ctx context.Context, name string) (context.Context, Span) {
	span := NewSimpleSpan(name)
	t.mu.Lock()
	t.spans = append(t.spans, span)
	t.mu.Unlock()
	return ctx, span
}

// GetSpans returns all spans recorded by this tracer.
func (t *SimpleTracer) GetSpans() []*SimpleSpan {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]*SimpleSpan, len(t.spans))
	copy(result, t.spans)
	return result
}

var (
	defaultTracer Tracer = &noopTracer{}
	tracerMu      sync.Mutex
)

// SetDefaultTracer sets the default tracer.
func SetDefaultTracer(tracer Tracer) {
	tracerMu.Lock()
	defaultTracer = tracer
	tracerMu.Unlock()
}

// GetDefaultTracer returns the default tracer.
func GetDefaultTracer() Tracer {
	tracerMu.Lock()
	defer tracerMu.Unlock()
	return defaultTracer
}

// StartSpan starts a new span using the default tracer.
func StartSpan(ctx context.Context, name string) (context.Context, Span) {
	return GetDefaultTracer().Start(ctx, name)
}
