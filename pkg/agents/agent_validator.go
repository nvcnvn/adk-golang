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

package agents

import (
	"fmt"
	"regexp"
	"strings"
)

// Reserved agent names that cannot be used
var reservedAgentNames = map[string]bool{
	"user": true,
}

// Valid Go identifier pattern
var validIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateAgentName validates that the agent name is a valid identifier and not reserved
func ValidateAgentName(name string) error {
	// Check if name is empty
	if name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}

	// Check if the name is a valid Go identifier
	if !validIdentifierPattern.MatchString(name) {
		return fmt.Errorf(
			"invalid agent name: '%s'. Agent name must be a valid identifier. "+
				"It should start with a letter (a-z, A-Z) or an underscore (_), "+
				"and can only contain letters, digits (0-9), and underscores",
			name,
		)
	}

	// Check if the name is a reserved word
	nameLower := strings.ToLower(name)
	if reservedAgentNames[nameLower] {
		return fmt.Errorf(
			"agent name cannot be '%s'. '%s' is reserved for end-user's input",
			name, nameLower,
		)
	}

	return nil
}

// ValidateAgentHierarchy validates parent-child relationships in the agent hierarchy
// It returns an error if an agent is already assigned to another parent
func ValidateAgentHierarchy(subAgent, newParent BaseAgent) error {
	// If subAgent doesn't implement ParentGetter, we can't check its parent
	if pg, ok := subAgent.(interface{ ParentAgent() BaseAgent }); ok {
		if parent := pg.ParentAgent(); parent != nil && parent != newParent {
			return fmt.Errorf(
				"agent '%s' already has a parent agent: '%s', cannot add to '%s'",
				subAgent.Name(), parent.Name(), newParent.Name(),
			)
		}
	}
	return nil
}
