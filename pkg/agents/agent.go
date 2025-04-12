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

// Package agents provides the core agent types and functionality.
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nvcnvn/adk-golang/pkg/models"
	"github.com/nvcnvn/adk-golang/pkg/telemetry"
	"github.com/nvcnvn/adk-golang/pkg/tools"
)

// BeforeAgentCallback is a function that's called before an agent processes a message
type BeforeAgentCallback func(ctx context.Context, message string) (string, bool)

// AfterAgentCallback is a function that's called after an agent processes a message
type AfterAgentCallback func(ctx context.Context, response string) string

// Agent represents an AI agent that can process user inputs and generate responses.
type Agent struct {
	name        string
	model       string
	instruction string
	description string
	tools       []tools.Tool
	subAgents   []*Agent
	parentAgent *Agent

	// Callbacks
	beforeAgentCallback BeforeAgentCallback
	afterAgentCallback  AfterAgentCallback

	// Additional fields that may be needed
	registry *agentRegistry
}

// Config holds configuration options for creating a new agent.
type Config struct {
	Name        string
	Model       string
	Instruction string
	Description string
	Tools       []tools.Tool
	SubAgents   []*Agent

	// Callbacks
	BeforeAgentCallback BeforeAgentCallback
	AfterAgentCallback  AfterAgentCallback
}

// Option defines a function type for configuring an agent.
type Option func(*Config)

// WithName sets the name of the agent.
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithModel sets the model to use for the agent.
func WithModel(model string) Option {
	return func(c *Config) {
		c.Model = model
	}
}

// WithInstruction sets the instruction for the agent.
func WithInstruction(instruction string) Option {
	return func(c *Config) {
		c.Instruction = instruction
	}
}

// WithDescription sets the description of the agent.
func WithDescription(description string) Option {
	return func(c *Config) {
		c.Description = description
	}
}

// WithTools sets the tools for the agent to use.
func WithTools(tools ...tools.Tool) Option {
	return func(c *Config) {
		c.Tools = tools
	}
}

// WithSubAgents sets the sub-agents for this agent.
func WithSubAgents(subAgents ...*Agent) Option {
	return func(c *Config) {
		c.SubAgents = subAgents
	}
}

// WithBeforeAgentCallback sets a callback that runs before agent processing.
func WithBeforeAgentCallback(callback BeforeAgentCallback) Option {
	return func(c *Config) {
		c.BeforeAgentCallback = callback
	}
}

// WithAfterAgentCallback sets a callback that runs after agent processing.
func WithAfterAgentCallback(callback AfterAgentCallback) Option {
	return func(c *Config) {
		c.AfterAgentCallback = callback
	}
}

// NewAgent creates a new agent with the provided options.
func NewAgent(options ...Option) *Agent {
	config := &Config{
		Model: "gemini-1.5-pro", // Default model
	}

	for _, option := range options {
		option(config)
	}

	agent := &Agent{
		name:                config.Name,
		model:               config.Model,
		instruction:         config.Instruction,
		description:         config.Description,
		tools:               config.Tools,
		subAgents:           config.SubAgents,
		beforeAgentCallback: config.BeforeAgentCallback,
		afterAgentCallback:  config.AfterAgentCallback,
	}

	// Set parent agent for sub-agents
	for _, subAgent := range agent.subAgents {
		if subAgent.parentAgent != nil {
			// Log warning but continue - in Go we'll allow multiple parents unlike Python
			telemetry.Logger.Printf("Warning: Agent %s already has a parent: %s",
				subAgent.name, subAgent.parentAgent.name)
		}
		subAgent.parentAgent = agent
	}

	return agent
}

// Process handles a user message and generates a response.
func (a *Agent) Process(ctx context.Context, message string) (string, error) {
	// Create a span for tracking this processing
	ctx, span := telemetry.StartSpan(ctx, "Agent.Process")
	defer span.End()

	span.SetAttribute("agent.name", a.name)
	span.SetAttribute("agent.model", a.model)
	span.SetAttribute("input.length", fmt.Sprintf("%d", len(message)))

	// Run before agent callback if present
	if a.beforeAgentCallback != nil {
		if result, skipProcessing := a.beforeAgentCallback(ctx, message); skipProcessing {
			return result, nil
		}
	}

	// Get the model from the registry
	modelRegistry := models.GetRegistry()
	model, ok := modelRegistry.Get(a.model)

	if !ok {
		// Fall back to mock implementation if model is not available
		span.SetAttribute("model.fallback", "true")
		return "I'm sorry, but the requested model is not available. (This is a placeholder response.)", nil
	}

	// Prepare messages
	msgs := []models.Message{
		{
			Role:    "system",
			Content: a.instruction,
		},
		{
			Role:    "user",
			Content: message,
		},
	}

	// If there are tools, add them to the system message
	if len(a.tools) > 0 {
		toolsJSON, err := json.Marshal(a.getToolDefinitions())
		if err == nil {
			msgs = append([]models.Message{
				{
					Role: "system",
					Content: fmt.Sprintf("%s\n\nYou have access to the following tools: %s",
						a.instruction, string(toolsJSON)),
				},
			}, msgs[1:]...)
		}
	}

	// Generate response
	response, err := model.Generate(ctx, msgs)
	if err != nil {
		span.SetAttribute("error", err.Error())
		return "", err
	}

	span.SetAttribute("output.length", fmt.Sprintf("%d", len(response)))

	// Run after agent callback if present
	if a.afterAgentCallback != nil {
		response = a.afterAgentCallback(ctx, response)
	}

	return response, nil
}

// RootAgent returns the root agent in the hierarchy
func (a *Agent) RootAgent() *Agent {
	root := a
	for root.parentAgent != nil {
		root = root.parentAgent
	}
	return root
}

// FindAgent searches for an agent by name in the agent tree
func (a *Agent) FindAgent(name string) *Agent {
	// Check if this is the agent we're looking for
	if a.name == name {
		return a
	}

	// Search in sub-agents
	return a.FindSubAgent(name)
}

// FindSubAgent searches for an agent by name in sub-agents
func (a *Agent) FindSubAgent(name string) *Agent {
	for _, subAgent := range a.subAgents {
		if found := subAgent.FindAgent(name); found != nil {
			return found
		}
	}
	return nil
}

// getToolDefinitions returns a JSON-serializable representation of the tools.
func (a *Agent) getToolDefinitions() []map[string]interface{} {
	var defs []map[string]interface{}

	for _, tool := range a.tools {
		schema := tool.Schema()
		def := map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"input":       schema.Input,
			"output":      schema.Output,
		}
		defs = append(defs, def)
	}

	return defs
}

// Name returns the name of the agent.
func (a *Agent) Name() string {
	return a.name
}

// Model returns the model used by the agent.
func (a *Agent) Model() string {
	return a.model
}

// Instruction returns the instruction for the agent.
func (a *Agent) Instruction() string {
	return a.instruction
}

// Description returns the description of the agent.
func (a *Agent) Description() string {
	return a.description
}

// Tools returns the tools available to the agent.
func (a *Agent) Tools() []tools.Tool {
	return a.tools
}

// SubAgents returns the sub-agents of this agent.
func (a *Agent) SubAgents() []*Agent {
	return a.subAgents
}

// ParentAgent returns the parent agent of this agent.
func (a *Agent) ParentAgent() *Agent {
	return a.parentAgent
}

// Agent registry to keep track of exported agents
type agentRegistry struct {
	agents map[string]*Agent
	mu     sync.RWMutex
}

var (
	registry     = &agentRegistry{agents: make(map[string]*Agent)}
	registryOnce sync.Once
)

// getRegistry returns the singleton agent registry.
func getRegistry() *agentRegistry {
	registryOnce.Do(func() {
		registry = &agentRegistry{
			agents: make(map[string]*Agent),
		}
	})
	return registry
}

// Register registers an agent with the registry.
func (r *agentRegistry) Register(name string, agent *Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[name] = agent
}

// Get returns an agent from the registry by name.
func (r *agentRegistry) Get(name string) (*Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[name]
	return agent, ok
}

// Export makes an agent available to the ADK CLI.
func Export(agent *Agent) {
	registry := getRegistry()
	registry.Register(agent.Name(), agent)
}

// GetExportedAgent retrieves an agent that was exported with Export.
func GetExportedAgent(name string) (*Agent, bool) {
	registry := getRegistry()
	return registry.Get(name)
}
