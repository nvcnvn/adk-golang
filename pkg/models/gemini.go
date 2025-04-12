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

package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	defaultGeminiAPIEndpoint = "https://generativelanguage.googleapis.com/v1beta"
)

// GeminiModel is an implementation of Model that uses the Gemini API.
type GeminiModel struct {
	BaseModel
	apiKey   string
	endpoint string
}

// GeminiRequest represents a request to the Gemini API.
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

// GeminiContent represents a content item in a Gemini request.
type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part in a Gemini content item.
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiResponse represents a response from the Gemini API.
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
	Error      *GeminiError      `json:"error,omitempty"`
}

// GeminiCandidate represents a candidate in a Gemini response.
type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

// GeminiError represents an error in a Gemini response.
type GeminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewGeminiModel creates a new GeminiModel with the given model name and API key.
func NewGeminiModel(modelName string) (*GeminiModel, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GOOGLE_API_KEY environment variable not set")
	}

	endpoint := os.Getenv("GEMINI_API_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultGeminiAPIEndpoint
	}

	return &GeminiModel{
		BaseModel: BaseModel{name: modelName},
		apiKey:    apiKey,
		endpoint:  endpoint,
	}, nil
}

// Generate generates a response to the given messages using the Gemini API.
func (m *GeminiModel) Generate(ctx context.Context, messages []Message) (string, error) {
	req, err := m.createRequest(messages)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", m.endpoint, m.name, m.apiKey)
	resp, err := m.sendRequest(ctx, url, req)
	if err != nil {
		return "", err
	}

	if resp.Error != nil {
		return "", fmt.Errorf("gemini API error: %d %s", resp.Error.Code, resp.Error.Message)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("no content in response")
	}

	return resp.Candidates[0].Content.Parts[0].Text, nil
}

// GenerateStream generates a streaming response to the given messages using the Gemini API.
func (m *GeminiModel) GenerateStream(ctx context.Context, messages []Message) (chan StreamedResponse, error) {
	// This is a simplified implementation. In a real implementation, you'd use the Gemini streaming API.
	// For now, we'll just generate a response and stream it character by character.

	ch := make(chan StreamedResponse)

	go func() {
		defer close(ch)

		response, err := m.Generate(ctx, messages)
		if err != nil {
			ch <- StreamedResponse{
				Error: err,
				Done:  true,
			}
			return
		}

		for i, char := range response {
			select {
			case <-ctx.Done():
				ch <- StreamedResponse{
					Error: ctx.Err(),
					Done:  true,
				}
				return
			default:
				ch <- StreamedResponse{
					Content: string(char),
					Done:    i == len(response)-1,
				}
			}
		}
	}()

	return ch, nil
}

// createRequest creates a Gemini API request from a list of messages.
func (m *GeminiModel) createRequest(messages []Message) (*GeminiRequest, error) {
	if len(messages) == 0 {
		return nil, errors.New("no messages provided")
	}

	var geminiContents []GeminiContent

	for _, msg := range messages {
		role := msg.Role
		// Convert to Gemini's role format
		if role == "user" {
			role = "user"
		} else if role == "assistant" {
			role = "model"
		} else if role == "system" {
			role = "user" // Gemini doesn't have a distinct system role, so we prepend it as a user message
		}

		content := GeminiContent{
			Role: role,
			Parts: []GeminiPart{
				{Text: msg.Content},
			},
		}

		geminiContents = append(geminiContents, content)
	}

	return &GeminiRequest{
		Contents: geminiContents,
	}, nil
}

// sendRequest sends a request to the Gemini API and returns the response.
func (m *GeminiModel) sendRequest(ctx context.Context, url string, req *GeminiRequest) (*GeminiResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var geminiResponse GeminiResponse
	err = json.Unmarshal(body, &geminiResponse)
	if err != nil {
		return nil, err
	}

	return &geminiResponse, nil
}

func init() {
	// Register the Gemini models
	geminiModels := []string{
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-2.0-pro",
		"gemini-2.0-flash",
	}

	for _, modelName := range geminiModels {
		model, err := NewGeminiModel(modelName)
		if err == nil {
			GetRegistry().Register(model)
		}
	}
}
