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

// ReadonlyContext provides a read-only view of an invocation context.
// This is used in situations where a component only needs to read, not modify, the context.
type ReadonlyContext interface {
	// GetID returns the invocation ID
	GetID() string

	// GetAgentName returns the name of the agent
	GetAgentName() string

	// IsEndInvocation returns whether the invocation should end
	IsEndInvocation() bool

	// GetTranscriptionCache returns the transcription cache
	GetTranscriptionCache() interface{}
}

// Ensure InvocationContext implements ReadonlyContext interface
var _ ReadonlyContext = (*InvocationContext)(nil)
