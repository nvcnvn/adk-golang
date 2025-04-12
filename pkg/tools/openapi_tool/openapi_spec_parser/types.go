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

// Package openapi_spec_parser provides functionality to parse OpenAPI specifications
// and generate tools from them.
package openapi_spec_parser

// OperationEndpoint represents an endpoint in an OpenAPI specification.
type OperationEndpoint struct {
	Path   string
	Method string
}

// ParsedOperation represents a parsed operation from an OpenAPI specification.
type ParsedOperation struct {
	ID          string
	Summary     string
	Description string
	Endpoint    OperationEndpoint
	Parameters  map[string]any
	RequestBody map[string]any
	Responses   map[string]any
	Tags        []string
}

// OpenAPISpecParser parses OpenAPI specifications.
type OpenAPISpecParser interface {
	ParseSpec(spec map[string]any) ([]ParsedOperation, error)
}
