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

/*
Package mcp_tool provides functionality for working with MCP (Model Context Protocol) tools.

This package enables integration with MCP servers, allowing ADK tools to interact with
MCP-compatible tools. It provides functionality to connect to MCP servers, list available
tools, and execute tool commands through an MCP session.

Example usage with stdio server:

	ctx := context.Background()

	// Connect to an MCP server using stdio
	params := mcp_tool.StdioServerParams{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
	}

	// One-step initialization and loading tools
	tools, closer, err := mcp_tool.FromServer(ctx, params)
	if err != nil {
		log.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer closer.Close()

	// Use the tools in an agent
	agent := agents.NewLlmAgent(
		agents.WithTools(tools...),
	)

Example usage with SSE server:

	ctx := context.Background()

	// Connect to an MCP server using SSE
	params := mcp_tool.SseServerParams{
		URL:            "http://0.0.0.0:8090/sse",
		Timeout:        5.0,
		SseReadTimeout: 300.0, // 5 minutes
	}

	// Step-by-step initialization and loading tools
	toolset := mcp_tool.NewMcpToolset(params)

	// Initialize the connection
	if err := toolset.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize MCP toolset: %v", err)
	}
	defer toolset.Close()

	// Load tools from the server
	tools, err := toolset.LoadTools(ctx)
	if err != nil {
		log.Fatalf("Failed to load MCP tools: %v", err)
	}

	// Use the tools in your agent
	agent := agents.NewLlmAgent(
		agents.WithTools(tools...),
	)

The package provides interfaces and implementations for the following components:

- McpTool: Represents a tool that can be executed through an MCP session.
- McpToolset: Manages the connection to an MCP server and provides tools.
- ConnectionParams: Base interface for different types of connection parameters.
- StdioServerParams: Parameters for connecting to an MCP server using stdio.
- SseServerParams: Parameters for connecting to an MCP server using SSE.
*/
package mcp_tool
