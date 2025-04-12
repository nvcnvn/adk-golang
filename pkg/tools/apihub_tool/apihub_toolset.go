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

// Package apihub_tool provides functionality to generate tools from API Hub resources.
package apihub_tool

import (
	"context"
	"fmt"
	"sync"

	"github.com/nvcnvn/adk-golang/pkg/auth"
	"github.com/nvcnvn/adk-golang/pkg/tools"
	"github.com/nvcnvn/adk-golang/pkg/tools/apihub_tool/clients"
	"github.com/nvcnvn/adk-golang/pkg/tools/openapi_tool/common"
	"github.com/nvcnvn/adk-golang/pkg/tools/openapi_tool/openapi_spec_parser"
	"gopkg.in/yaml.v3"
)

// APIHubToolset generates tools from a given API Hub resource.
type APIHubToolset struct {
	ctx                context.Context
	apihubResourceName string
	name               string
	description        string
	lazyLoadSpec       bool
	apihubClient       clients.BaseAPIHubClient
	generatedTools     map[string]tools.Tool
	authScheme         auth.AuthScheme
	authCredential     auth.AuthCredential
	mu                 sync.RWMutex
}

// APIHubToolsetOption defines options for creating a new APIHubToolset.
type APIHubToolsetOption struct {
	// Parameters for fetching API Hub resource
	ApihubResourceName string
	AccessToken        string
	ServiceAccountJSON string

	// Parameters for the toolset itself
	Name        string
	Description string

	// Parameters for generating tools
	LazyLoadSpec   bool
	AuthScheme     auth.AuthScheme
	AuthCredential auth.AuthCredential

	// Optionally, you can provide a custom API Hub client
	ApihubClient clients.BaseAPIHubClient
}

// NewAPIHubToolset creates a new APIHubToolset with the provided options.
func NewAPIHubToolset(ctx context.Context, opt APIHubToolsetOption) (*APIHubToolset, error) {
	var apihubClient clients.BaseAPIHubClient

	if opt.ApihubClient != nil {
		apihubClient = opt.ApihubClient
	} else {
		clientOpt := clients.APIHubClientOption{
			AccessToken:        opt.AccessToken,
			ServiceAccountJSON: opt.ServiceAccountJSON,
		}
		apiHubClient, err := clients.NewAPIHubClient(ctx, clientOpt)
		if err != nil {
			return nil, fmt.Errorf("failed to create API Hub client: %w", err)
		}
		apihubClient = apiHubClient
	}

	toolset := &APIHubToolset{
		ctx:                ctx,
		apihubResourceName: opt.ApihubResourceName,
		name:               opt.Name,
		description:        opt.Description,
		lazyLoadSpec:       opt.LazyLoadSpec,
		apihubClient:       apihubClient,
		authScheme:         opt.AuthScheme,
		authCredential:     opt.AuthCredential,
		generatedTools:     make(map[string]tools.Tool),
	}

	if !opt.LazyLoadSpec {
		if err := toolset.prepareTools(); err != nil {
			return nil, fmt.Errorf("failed to prepare tools: %w", err)
		}
	}

	return toolset, nil
}

// GetTool retrieves a specific tool by its name.
func (t *APIHubToolset) GetTool(name string) (tools.Tool, bool) {
	if !t.areToolsReady() {
		if err := t.prepareTools(); err != nil {
			return nil, false
		}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	tool, ok := t.generatedTools[name]
	return tool, ok
}

// GetTools retrieves all available tools.
func (t *APIHubToolset) GetTools() []tools.Tool {
	if !t.areToolsReady() {
		if err := t.prepareTools(); err != nil {
			return nil
		}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	tools := make([]tools.Tool, 0, len(t.generatedTools))
	for _, tool := range t.generatedTools {
		tools = append(tools, tool)
	}

	return tools
}

// areToolsReady checks if the tools have been generated.
func (t *APIHubToolset) areToolsReady() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return !t.lazyLoadSpec || len(t.generatedTools) > 0
}

// prepareTools fetches the spec from API Hub and generates the tools.
func (t *APIHubToolset) prepareTools() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get the spec content from API Hub
	specContent, err := t.apihubClient.GetSpecContent(t.ctx, t.apihubResourceName)
	if err != nil {
		return fmt.Errorf("failed to get spec content: %w", err)
	}

	// Parse the spec and generate tools
	generatedTools, err := t.parseSpecToTools(specContent)
	if err != nil {
		return fmt.Errorf("failed to parse spec to tools: %w", err)
	}

	// Store the generated tools
	for _, tool := range generatedTools {
		t.generatedTools[tool.Name()] = tool
	}

	return nil
}

// parseSpecToTools parses the spec string to a list of tools.
func (t *APIHubToolset) parseSpecToTools(specStr string) ([]tools.Tool, error) {
	// Parse YAML spec
	var specDict map[string]interface{}
	if err := yaml.Unmarshal([]byte(specStr), &specDict); err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	// If spec is empty, return empty slice
	if len(specDict) == 0 {
		return nil, nil
	}

	// Extract API info
	info, ok := specDict["info"].(map[string]interface{})
	if ok {
		if t.name == "" {
			if title, ok := info["title"].(string); ok {
				t.name = common.ToSnakeCase(title)
			} else {
				t.name = "unnamed"
			}
		}

		if t.description == "" {
			if desc, ok := info["description"].(string); ok {
				t.description = desc
			}
		}
	}

	// Create OpenAPIToolset to generate tools
	openAPIToolset := openapi_spec_parser.NewOpenAPIToolset(
		specDict,
		t.authScheme,
		t.authCredential,
	)

	return openAPIToolset.GetTools(), nil
}
