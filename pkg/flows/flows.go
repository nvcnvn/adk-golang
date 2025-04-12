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

// Package flows provides workflow implementations for agent processing
package flows

import (
	"fmt"

	"github.com/nvcnvn/adk-golang/pkg/code_executors"
	"github.com/nvcnvn/adk-golang/pkg/flows/llm_flows"
)

// CreateBasicFlow creates a new basic LLM flow with standard processors
func CreateBasicFlow() *llm_flows.BasicFlow {
	flow := llm_flows.NewBasicFlow()

	// Add standard request processors
	flow.RequestProcessors = append(flow.RequestProcessors,
		llm_flows.NewInstructionsProcessor(),
		llm_flows.NewContentsProcessor(),
	)

	return flow
}

// CreateIdentityFlow creates a minimal identity flow with no processors
func CreateIdentityFlow() *llm_flows.IdentityFlow {
	return llm_flows.NewIdentityFlow()
}

// CreateCodeExecutionFlow creates a flow with code execution capabilities
func CreateCodeExecutionFlow(codeExecutorName string) (*llm_flows.BasicFlow, error) {
	flow := CreateBasicFlow()

	// Create and add code execution processor
	codeExecutor, err := CreateCodeExecutor(codeExecutorName)
	if err != nil {
		return nil, err
	}

	codeExecutionProcessor := llm_flows.NewCodeExecutionProcessor(codeExecutor)
	flow.ResponseProcessors = append(flow.ResponseProcessors, codeExecutionProcessor)

	return flow, nil
}

// CreateCodeExecutor creates a code executor by name
func CreateCodeExecutor(name string) (code_executors.CodeExecutor, error) {
	// This would be implemented in the code_executors package
	// For now it's a placeholder that would return the appropriate executor
	return nil, fmt.Errorf("code executor %s not implemented yet", name)
}
