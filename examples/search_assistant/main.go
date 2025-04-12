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

// Package main provides an example search assistant agent implementation.
package main

import (
	"fmt"
	"os"

	"github.com/nvcnvn/adk-golang/pkg/agents"
	"github.com/nvcnvn/adk-golang/pkg/tools"
)

func main() {
	// Define the search assistant agent
	searchAgent := agents.NewAgent(
		agents.WithName("search_assistant"),
		agents.WithModel("gemini-2.0-flash-exp"), // Or your preferred Gemini model
		agents.WithInstruction("You are a helpful assistant. Answer user questions using Google Search when needed."),
		agents.WithDescription("An assistant that can search the web."),
		agents.WithTools(tools.GoogleSearch),
	)

	// Export the agent for the CLI to use
	agents.Export(searchAgent)

	fmt.Println("Search agent exported. You can now run it with:")
	fmt.Println("adk run search_assistant")

	// This part would normally not be needed in a real application,
	// as the ADK would handle the agent registration and execution.
	// It's here for demonstration purposes.
	if len(os.Args) > 1 && os.Args[1] == "run" {
		fmt.Println("Starting interactive session with search assistant")
		fmt.Println("Type your query or 'exit' to quit")

		for {
			fmt.Print("> ")
			var input string
			fmt.Scanln(&input)

			if input == "exit" {
				break
			}

			// For demo purposes, just echo back a mock response
			fmt.Println("Agent: I would search for information about:", input)
			fmt.Println("(This is a placeholder. The actual implementation would use the Google Search tool)")
		}
	}
}
