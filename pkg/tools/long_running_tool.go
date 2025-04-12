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

package tools

// NewLongRunningFunctionTool creates a tool wrapper for a function that takes a long time to complete.
// The function will be executed as usual, but the LLM will be informed that this is a long-running
// operation and should not wait for an immediate response. The result will be returned asynchronously.
//
// This is useful for operations that may take significant time, like complex calculations,
// large data processing, or operations requiring user interaction.
func NewLongRunningFunctionTool(fn interface{}, config FunctionToolConfig) (*LlmToolAdaptor, error) {
	// Force isLongRunning to be true
	config.IsLongRunning = true

	// Create a function tool with the long-running flag set
	return NewFunctionTool(fn, config)
}

// ConvertToLongRunning converts an existing tool to a long-running tool.
// This is useful when you want to make an existing tool long-running without
// recreating it.
func ConvertToLongRunning(tool Tool) *LlmToolAdaptor {
	return NewLlmToolAdaptor(tool, true)
}
