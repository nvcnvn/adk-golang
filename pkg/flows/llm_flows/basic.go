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
)

// BasicFlow is a simple implementation of BaseLlmFlow
type BasicFlow struct {
	*BaseLlmFlow
}

// NewBasicFlow creates a new BasicFlow instance
func NewBasicFlow() *BasicFlow {
	return &BasicFlow{
		BaseLlmFlow: NewBaseLlmFlow(),
	}
}

// Run executes the basic flow with the given invocation context
func (f *BasicFlow) Run(ctx context.Context, invocationContext *agents.InvocationContext) (<-chan *events.Event, error) {
	return f.BaseLlmFlow.Run(ctx, invocationContext)
}
