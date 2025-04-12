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

// Package planners provides planner interfaces and implementations for ADK.
// Planners allow agents to generate plans for queries to guide their action.
package planners

import "github.com/nvcnvn/adk-golang/pkg/models"

// This file exposes the core planner types and constructors
// for easy import and use by other packages.

// NewDefaultBuiltInPlanner creates a BuiltInPlanner with default thinking configuration.
func NewDefaultBuiltInPlanner() *BuiltInPlanner {
	return NewBuiltInPlanner(models.NewThinkingConfig())
}
