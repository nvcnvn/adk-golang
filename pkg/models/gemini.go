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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
)

const (
	defaultGeminiAPIEndpoint = "https://generativelanguage.googleapis.com/v1"
)

// GeminiLLM implements the LLM interface for Google's Gemini models.
type GeminiLLM struct {
	ModelName string
	apiKey    string
	endpoint  string
	client    *http.Client
}

// geminiRequest represents a request to the Gemini API
type geminiRequest struct {
	Contents          []geminiContent        `json:"contents"`
	SystemInstruction string                 `json:"systemInstruction,omitempty"`
	Tools             []geminiTool           `json:"tools,omitempty"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig,omitempty"`
}

// geminiContent represents a message with role and parts in the Gemini API format
type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

// geminiPart represents a part of content in the Gemini API format
type geminiPart struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inlineData,omitempty"`
}

// inlineData represents inline binary data with MIME type
type inlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// geminiTool represents a tool in the Gemini API format
type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

// geminiFunctionDeclaration represents a function declaration in Gemini format
type geminiFunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// geminiGenerationConfig represents generation config for Gemini requests
type geminiGenerationConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	TopK            int      `json:"topK,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// geminiResponse represents a response from the Gemini API
type geminiResponse struct {
	Candidates     []geminiCandidate `json:"candidates"`
	PromptFeedback *promptFeedback   `json:"promptFeedback,omitempty"`
}

// geminiCandidate represents a candidate in a Gemini response
type geminiCandidate struct {
	Content       geminiContent  `json:"content"`
	FinishReason  string         `json:"finishReason,omitempty"`
	Index         int            `json:"index"`
	SafetyRatings []safetyRating `json:"safetyRatings,omitempty"`
}

// safetyRating represents a safety rating in a Gemini response
type safetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// promptFeedback represents feedback on the prompt in a Gemini response
type promptFeedback struct {
	SafetyRatings []safetyRating `json:"safetyRatings,omitempty"`
}

// NewGeminiLLM creates a new Gemini LLM client.
func NewGeminiLLM(modelName string) (*GeminiLLM, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GOOGLE_API_KEY environment variable not set")
	}

	endpoint := os.Getenv("GEMINI_API_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultGeminiAPIEndpoint
	}

	return &GeminiLLM{
		ModelName: modelName,
		apiKey:    apiKey,
		endpoint:  endpoint,
		client:    &http.Client{},
	}, nil
}

// SupportedModels returns a list of regex patterns for models supported by Gemini.
func (g *GeminiLLM) SupportedModels() []string {
	return []string{
		`gemini-.*`,
		`projects\/.*\/locations\/.*\/endpoints\/.*`,
		`projects\/.*\/locations\/.*\/publishers\/google\/models\/gemini.*`,
	}
}

// GenerateContent generates content based on the provided request.
func (g *GeminiLLM) GenerateContent(ctx context.Context, request *LlmRequest) (*LlmResponse, error) {
	geminiReq, err := g.createGeminiRequest(request)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		g.endpoint, g.ModelName, g.apiKey)

	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-client", g.getUserAgent())

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return g.createResponse(&geminiResp), nil
}

// GenerateContentStream generates streaming content based on the provided request.
func (g *GeminiLLM) GenerateContentStream(ctx context.Context, request *LlmRequest) (<-chan *LlmResponse, error) {
	geminiReq, err := g.createGeminiRequest(request)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s",
		g.endpoint, g.ModelName, g.apiKey)

	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-client", g.getUserAgent())

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	responseChan := make(chan *LlmResponse)

	go func() {
		defer resp.Body.Close()
		defer close(responseChan)

		decoder := json.NewDecoder(resp.Body)
		fullText := ""

		for {
			select {
			case <-ctx.Done():
				return
			default:
				var chunk geminiResponse
				if err := decoder.Decode(&chunk); err != nil {
					if err != io.EOF {
						// Send error only if it's not EOF
						errResp := &LlmResponse{
							ErrorMessage: fmt.Sprintf("Error: %v", err),
						}
						responseChan <- errResp
					}
					return
				}

				// Convert the chunk to an LlmResponse
				llmResp := g.createResponse(&chunk)

				// Handle streaming text aggregation
				if text := g.extractTextFromResponse(llmResp); text != "" {
					fullText += text

					// Mark as partial unless it's the final chunk
					if len(chunk.Candidates) > 0 && chunk.Candidates[0].FinishReason != "" {
						// Send the accumulated text as the final non-partial message
						finalResp := &LlmResponse{
							Content: &Content{
								Parts: []*Part{{
									Text: fullText,
									Role: "assistant",
								}},
							},
							Partial: false,
						}
						responseChan <- finalResp
					} else {
						// This is a partial response
						llmResp.Partial = true
						responseChan <- llmResp
					}
				} else {
					responseChan <- llmResp
				}
			}
		}
	}()

	return responseChan, nil
}

// Connect establishes a real-time connection with the model.
func (g *GeminiLLM) Connect(ctx context.Context, request *LlmRequest) (LlmConnection, error) {
	// This would be implemented with the Gemini API's bidirectional streaming
	// For now, return a not implemented error
	return nil, errors.New("bidirectional connection not implemented")
}

// createGeminiRequest converts LlmRequest to geminiRequest
func (g *GeminiLLM) createGeminiRequest(request *LlmRequest) (*geminiRequest, error) {
	// Create geminiContent array from existing message structure
	var contents []geminiContent

	// Handle system instructions if specified
	if request.SystemInstructions != "" {
		// Add system instructions as a user message
		contents = append(contents, geminiContent{
			Role: "user",
			Parts: []geminiPart{{
				Text: request.SystemInstructions,
			}},
		})
	}

	// Convert contents from message parts
	if request.Contents != nil && len(request.Contents.Parts) > 0 {
		for _, part := range request.Contents.Parts {
			var role string
			if part.Role == "" {
				role = "user" // Default role
			} else {
				role = part.Role
			}

			geminiParts := []geminiPart{{Text: part.Text}}

			contents = append(contents, geminiContent{
				Role:  role,
				Parts: geminiParts,
			})
		}
	}

	// If no content was provided, add a default user message
	if len(contents) == 0 {
		contents = append(contents, geminiContent{
			Role: "user",
			Parts: []geminiPart{{
				Text: "Hello",
			}},
		})
	}

	// Ensure the last message is from the user so Gemini responds
	if len(contents) > 0 && contents[len(contents)-1].Role != "user" {
		contents = append(contents, geminiContent{
			Role: "user",
			Parts: []geminiPart{{
				Text: "Continue processing previous requests as instructed.",
			}},
		})
	}

	// Convert tools if present
	var tools []geminiTool
	if request.Tools != nil && len(request.Tools) > 0 {
		for _, tool := range request.Tools {
			geminiTool := geminiTool{
				FunctionDeclarations: []geminiFunctionDeclaration{
					{
						Name:        tool.Name,
						Description: tool.Description,
						Parameters:  tool.InputSchema,
					},
				},
			}
			tools = append(tools, geminiTool)
		}
	}

	return &geminiRequest{
		Contents:          contents,
		SystemInstruction: request.SystemInstructions,
		Tools:             tools,
		GenerationConfig: geminiGenerationConfig{
			Temperature:     request.Temperature,
			TopP:            request.TopP,
			TopK:            request.TopK,
			MaxOutputTokens: request.MaxTokens,
		},
	}, nil
}

// createResponse converts geminiResponse to LlmResponse
func (g *GeminiLLM) createResponse(geminiResp *geminiResponse) *LlmResponse {
	response := &LlmResponse{}

	if len(geminiResp.Candidates) > 0 {
		candidate := geminiResp.Candidates[0]
		content := &Content{
			Parts: make([]*Part, len(candidate.Content.Parts)),
		}

		for i, part := range candidate.Content.Parts {
			content.Parts[i] = &Part{
				Text: part.Text,
				Role: candidate.Content.Role,
			}
		}

		response.Content = content

		// Set finish reason
		switch candidate.FinishReason {
		case "STOP":
			// Normal completion
		case "MAX_TOKENS":
			response.ErrorMessage = "Response exceeded maximum token limit"
		case "SAFETY":
			response.ErrorMessage = "Response filtered due to safety concerns"
		case "RECITATION":
			response.ErrorMessage = "Response filtered due to recitation concerns"
		}
	}

	return response
}

// extractTextFromResponse gets text from LlmResponse
func (g *GeminiLLM) extractTextFromResponse(resp *LlmResponse) string {
	if resp.Content != nil && len(resp.Content.Parts) > 0 {
		for _, part := range resp.Content.Parts {
			if part.Text != "" {
				return part.Text
			}
		}
	}
	return ""
}

// getUserAgent returns a user agent string for API tracking
func (g *GeminiLLM) getUserAgent() string {
	// Create a tracking header similar to the Python version
	return fmt.Sprintf("google-adk/1.0 gl-go/%s", runtime.Version())
}

// NewGeminiModel creates a Gemini model with the standard Model interface
func NewGeminiModel(modelName string) (Model, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GOOGLE_API_KEY environment variable not set")
	}

	return &GeminiModel{
		BaseModel: BaseModel{name: modelName},
		apiKey:    apiKey,
		endpoint:  defaultGeminiAPIEndpoint,
	}, nil
}

// GeminiModel implements the Model interface for Gemini
type GeminiModel struct {
	BaseModel
	apiKey   string
	endpoint string
}

// Generate implements the Model interface
func (m *GeminiModel) Generate(ctx context.Context, messages []Message) (string, error) {
	// Convert Messages to geminiRequest
	contents := make([]geminiContent, len(messages))
	for i, msg := range messages {
		role := msg.Role
		if role == "system" {
			role = "user" // Gemini doesn't support system role directly
		}

		contents[i] = geminiContent{
			Role: role,
			Parts: []geminiPart{
				{Text: msg.Content},
			},
		}
	}

	req := &geminiRequest{
		Contents: contents,
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		m.endpoint, m.name, m.apiKey)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var result geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Candidates) == 0 {
		return "", errors.New("no candidates in response")
	}

	if len(result.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("no parts in candidate content")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

// GenerateStream implements streaming for the Model interface
func (m *GeminiModel) GenerateStream(ctx context.Context, messages []Message) (chan StreamedResponse, error) {
	// Convert Messages to geminiRequest
	contents := make([]geminiContent, len(messages))
	for i, msg := range messages {
		role := msg.Role
		if role == "system" {
			role = "user" // Gemini doesn't support system role directly
		}

		contents[i] = geminiContent{
			Role: role,
			Parts: []geminiPart{
				{Text: msg.Content},
			},
		}
	}

	req := &geminiRequest{
		Contents: contents,
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s",
		m.endpoint, m.name, m.apiKey)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	streamChan := make(chan StreamedResponse)

	go func() {
		defer resp.Body.Close()
		defer close(streamChan)

		decoder := json.NewDecoder(resp.Body)

		for {
			select {
			case <-ctx.Done():
				streamChan <- StreamedResponse{
					Error: ctx.Err(),
					Done:  true,
				}
				return
			default:
				var chunk geminiResponse
				if err := decoder.Decode(&chunk); err != nil {
					if err != io.EOF {
						// Send error only if it's not EOF
						streamChan <- StreamedResponse{
							Error: err,
							Done:  true,
						}
					}
					return
				}

				if len(chunk.Candidates) == 0 || len(chunk.Candidates[0].Content.Parts) == 0 {
					continue
				}

				isLast := chunk.Candidates[0].FinishReason != ""
				text := chunk.Candidates[0].Content.Parts[0].Text

				streamChan <- StreamedResponse{
					Content: text,
					Done:    isLast,
				}

				if isLast {
					return
				}
			}
		}
	}()

	return streamChan, nil
}

func init() {
	// Register with the enhanced registry
	registry := GetEnhancedRegistry()

	// Register model factory for Gemini
	for _, pattern := range []string{
		`gemini-.*`,
		`projects\/.*\/locations\/.*\/endpoints\/.*`,
		`projects\/.*\/locations\/.*\/publishers\/google\/models\/gemini.*`,
	} {
		err := registry.RegisterPattern(pattern, func(modelName string) (Model, error) {
			return NewGeminiModel(modelName)
		})
		if err != nil {
			// Log error but continue
			fmt.Printf("Error registering Gemini pattern %s: %v\n", pattern, err)
		}
	}
}
