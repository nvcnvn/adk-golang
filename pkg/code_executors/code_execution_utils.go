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

// Package code_executors provides functionality for executing code snippets.
package code_executors

import (
	"encoding/base64"
	"regexp"
	"strings"
)

// CodeExecutionUtils provides utility functions for code execution
type CodeExecutionUtils struct{}

// GetEncodedFileContent returns the base64-encoded content of a file
func (u *CodeExecutionUtils) GetEncodedFileContent(data []byte) string {
	// Check if data is already base64 encoded
	if isBase64Encoded(data) {
		return string(data)
	}
	return base64.StdEncoding.EncodeToString(data)
}

// isBase64Encoded checks if the data is already base64 encoded
func isBase64Encoded(data []byte) bool {
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return false
	}

	// Re-encode and compare
	reEncoded := base64.StdEncoding.EncodeToString(decoded)
	return string(data) == reEncoded
}

// ExtractCodeAndTruncateContent extracts the first code block from content and truncates everything after it
func (u *CodeExecutionUtils) ExtractCodeAndTruncateContent(
	content string,
	codeBlockDelimiters []CodeBlockDelimiter,
) string {
	if content == "" {
		return ""
	}

	// Build a regex pattern to find the first code block
	var leadingDelimiters []string
	var trailingDelimiters []string

	for _, delimiter := range codeBlockDelimiters {
		// Escape special regex characters
		leadingDelim := regexp.QuoteMeta(delimiter.Start)
		trailingDelim := regexp.QuoteMeta(delimiter.End)

		leadingDelimiters = append(leadingDelimiters, leadingDelim)
		trailingDelimiters = append(trailingDelimiters, trailingDelim)
	}

	leadingPattern := strings.Join(leadingDelimiters, "|")
	trailingPattern := strings.Join(trailingDelimiters, "|")

	pattern := regexp.MustCompile(`(?s)(?P<prefix>.*?)(?:` + leadingPattern + `)(?P<code>.*?)(?:` + trailingPattern + `)(?P<suffix>.*?)$`)

	match := pattern.FindStringSubmatch(content)
	if match == nil {
		return ""
	}

	// Find the index of the named group "code"
	codeIndex := -1
	for i, name := range pattern.SubexpNames() {
		if name == "code" {
			codeIndex = i
			break
		}
	}

	if codeIndex > 0 && codeIndex < len(match) {
		return match[codeIndex]
	}

	return ""
}

// FormatCodeBlock formats a code block with the given delimiters
func (u *CodeExecutionUtils) FormatCodeBlock(code string, delimiter CodeBlockDelimiter) string {
	return delimiter.Start + code + delimiter.End
}

// FormatExecutionResult formats an execution result with the given delimiters
func (u *CodeExecutionUtils) FormatExecutionResult(result *ExecutionResult, delimiter ExecutionResultDelimiter) string {
	var output strings.Builder

	output.WriteString(delimiter.Start)

	if result.Stderr != "" {
		output.WriteString("Error: ")
		output.WriteString(result.Stderr)
	} else {
		output.WriteString("Code execution result:\n")
		output.WriteString(result.Stdout)

		if len(result.OutputFiles) > 0 {
			output.WriteString("\n\nSaved artifacts:\n")
			var fileNames []string
			for _, file := range result.OutputFiles {
				fileNames = append(fileNames, "`"+file.Name+"`")
			}
			output.WriteString(strings.Join(fileNames, ","))
		}
	}

	output.WriteString(delimiter.End)

	return output.String()
}
