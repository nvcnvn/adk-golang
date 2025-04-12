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

// Package common provides common utilities for working with OpenAPI specifications.
package common

import (
	"regexp"
	"strings"
	"unicode"
)

// ToSnakeCase converts a string to snake_case.
func ToSnakeCase(s string) string {
	var result strings.Builder
	var prev rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && prev != '_' && !unicode.IsUpper(prev) {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
		prev = r
	}

	// Clean up consecutive underscores and trim
	output := result.String()
	output = regexp.MustCompile(`_+`).ReplaceAllString(output, "_")
	output = strings.Trim(output, "_")

	return output
}
