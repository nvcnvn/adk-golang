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

package mcp_tool

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// ConnectionParams is the base interface for MCP server connection parameters.
type ConnectionParams interface {
	// Type returns the type of connection parameters.
	Type() string
}

// StdioServerParams defines parameters for connecting to an MCP server using stdio.
type StdioServerParams struct {
	Command string   // The command to run the MCP server
	Args    []string // Arguments to the command
}

// Type returns the type of connection parameters.
func (p StdioServerParams) Type() string {
	return "stdio"
}

// SseServerParams defines parameters for connecting to an MCP server using SSE.
type SseServerParams struct {
	URL            string            // The URL of the SSE endpoint
	Headers        map[string]string // Headers to include in the request
	Timeout        float64           // Connection timeout in seconds
	SseReadTimeout float64           // SSE read timeout in seconds
}

// Type returns the type of connection parameters.
func (p SseServerParams) Type() string {
	return "sse"
}

// ClientFactory is an interface for creating MCP client sessions.
type ClientFactory interface {
	CreateClient(ctx context.Context, params ConnectionParams) (ClientSession, io.Closer, error)
}

// DefaultClientFactory is the default implementation of ClientFactory.
type DefaultClientFactory struct{}

// CreateClient creates a new MCP client session based on the provided parameters.
func (f DefaultClientFactory) CreateClient(ctx context.Context, params ConnectionParams) (ClientSession, io.Closer, error) {
	// TODO: Implement actual client creation based on the type of connection parameters.
	// This requires actual MCP client libraries to be implemented or imported.
	return nil, nil, errors.New("MCP client creation not implemented")
}

// McpToolset manages the connection to an MCP server and provides access to MCP tools.
type McpToolset struct {
	connectionParams ConnectionParams
	clientFactory    ClientFactory
	session          ClientSession
	closer           io.Closer
	initialized      bool
}

// NewMcpToolset creates a new MCP toolset.
func NewMcpToolset(params ConnectionParams) *McpToolset {
	return &McpToolset{
		connectionParams: params,
		clientFactory:    DefaultClientFactory{},
	}
}

// WithClientFactory sets the client factory to use.
func (m *McpToolset) WithClientFactory(factory ClientFactory) *McpToolset {
	m.clientFactory = factory
	return m
}

// Initialize initializes the connection to the MCP server.
func (m *McpToolset) Initialize(ctx context.Context) error {
	if m.initialized {
		return nil
	}

	if m.connectionParams == nil {
		return errors.New("connection parameters cannot be nil")
	}

	session, closer, err := m.clientFactory.CreateClient(ctx, m.connectionParams)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	m.session = session
	m.closer = closer

	// Initialize the session
	if err := m.session.Initialize(ctx); err != nil {
		_ = closer.Close() // Attempt to close but ignore errors
		return fmt.Errorf("failed to initialize MCP session: %w", err)
	}

	m.initialized = true
	return nil
}

// Close closes the connection to the MCP server.
func (m *McpToolset) Close() error {
	if !m.initialized || m.closer == nil {
		return nil
	}

	err := m.closer.Close()
	m.initialized = false
	m.session = nil
	m.closer = nil
	return err
}

// LoadTools loads all tools from the MCP server.
func (m *McpToolset) LoadTools(ctx context.Context) ([]McpTool, error) {
	if !m.initialized {
		return nil, errors.New("toolset not initialized, call Initialize first")
	}

	result, err := m.session.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	tools := make([]McpTool, 0, len(result.Tools))
	for _, tool := range result.Tools {
		// We can pass nil for both auth scheme and auth credential
		mcpTool, err := NewMcpTool(tool, m.session, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create MCP tool: %w", err)
		}
		tools = append(tools, *mcpTool)
	}

	return tools, nil
}

// FromServer is a convenience function to initialize a toolset and load tools in one step.
func FromServer(ctx context.Context, params ConnectionParams) ([]McpTool, io.Closer, error) {
	toolset := NewMcpToolset(params)

	if err := toolset.Initialize(ctx); err != nil {
		return nil, nil, err
	}

	tools, err := toolset.LoadTools(ctx)
	if err != nil {
		_ = toolset.Close() // Attempt to close but ignore errors
		return nil, nil, err
	}

	return tools, toolset, nil
}
