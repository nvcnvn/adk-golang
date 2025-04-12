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

package clients

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2/google"
)

// BaseAPIHubClient defines the interface for API Hub clients.
type BaseAPIHubClient interface {
	GetSpecContent(ctx context.Context, resourceName string) (string, error)
}

// APIHubClient is a client for interacting with the API Hub service.
type APIHubClient struct {
	rootURL        string
	accessToken    string
	serviceAccount string
	httpClient     *http.Client
}

// APIHubClientOption represents options for creating an APIHubClient.
type APIHubClientOption struct {
	AccessToken        string
	ServiceAccountJSON string
}

// NewAPIHubClient creates a new APIHubClient with the provided options.
func NewAPIHubClient(ctx context.Context, opt APIHubClientOption) (*APIHubClient, error) {
	client := &APIHubClient{
		rootURL:        "https://apihub.googleapis.com/v1",
		accessToken:    opt.AccessToken,
		serviceAccount: opt.ServiceAccountJSON,
		httpClient:     &http.Client{},
	}

	return client, nil
}

// GetSpecContent retrieves the specification content from API Hub based on the provided resource name.
func (c *APIHubClient) GetSpecContent(ctx context.Context, path string) (string, error) {
	apiResourceName, apiVersionResourceName, apiSpecResourceName, err := c.extractResourceName(path)
	if err != nil {
		return "", fmt.Errorf("failed to extract resource name: %w", err)
	}

	if apiResourceName != "" && apiVersionResourceName == "" {
		api, err := c.getAPI(ctx, apiResourceName)
		if err != nil {
			return "", fmt.Errorf("failed to get API: %w", err)
		}

		versions, ok := api["versions"].([]interface{})
		if !ok || len(versions) == 0 {
			return "", fmt.Errorf("no versions found in API Hub resource: %s", apiResourceName)
		}
		apiVersionResourceName = versions[0].(string)
	}

	if apiVersionResourceName != "" && apiSpecResourceName == "" {
		apiVersion, err := c.getAPIVersion(ctx, apiVersionResourceName)
		if err != nil {
			return "", fmt.Errorf("failed to get API version: %w", err)
		}

		specs, ok := apiVersion["specs"].([]interface{})
		if !ok || len(specs) == 0 {
			return "", fmt.Errorf("no specs found in API Hub version: %s", apiVersionResourceName)
		}
		apiSpecResourceName = specs[0].(string)
	}

	if apiSpecResourceName != "" {
		specContent, err := c.fetchSpec(ctx, apiSpecResourceName)
		if err != nil {
			return "", fmt.Errorf("failed to fetch spec: %w", err)
		}
		return specContent, nil
	}

	return "", fmt.Errorf("no API Hub resource found in path: %s", path)
}

// ListAPIs lists all APIs in the specified project and location.
func (c *APIHubClient) ListAPIs(ctx context.Context, project, location string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/apis", c.rootURL, project, location)

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		APIs []map[string]interface{} `json:"apis"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.APIs, nil
}

// getAPI gets details for a specific API.
func (c *APIHubClient) getAPI(ctx context.Context, apiResourceName string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s", c.rootURL, apiResourceName)

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// getAPIVersion gets details for a specific API version.
func (c *APIHubClient) getAPIVersion(ctx context.Context, apiVersionName string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s", c.rootURL, apiVersionName)

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// fetchSpec retrieves the content of a specific API specification.
func (c *APIHubClient) fetchSpec(ctx context.Context, apiSpecResourceName string) (string, error) {
	url := fmt.Sprintf("%s/%s:contents", c.rootURL, apiSpecResourceName)

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Contents string `json:"contents"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Contents == "" {
		return "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(result.Contents)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 content: %w", err)
	}

	return string(decoded), nil
}

// extractResourceName extracts API, version, and spec resource names from a URL or path.
func (c *APIHubClient) extractResourceName(urlOrPath string) (string, string, string, error) {
	var path string
	var queryParams url.Values

	// Try parsing as URL
	parsedURL, err := url.Parse(urlOrPath)
	if err == nil && parsedURL.Scheme != "" {
		path = parsedURL.Path
		queryParams = parsedURL.Query()

		// If path from UI, remove unnecessary prefix
		if strings.Contains(path, "api-hub/") {
			parts := strings.Split(path, "api-hub")
			if len(parts) > 1 {
				path = parts[1]
			}
		}
	} else {
		path = urlOrPath
	}

	// Split path into segments
	var pathSegments []string
	for _, segment := range strings.Split(path, "/") {
		if segment != "" {
			pathSegments = append(pathSegments, segment)
		}
	}

	var project, location, apiID, versionID, specID string

	// Extract project ID
	projectIdx := -1
	for i, segment := range pathSegments {
		if segment == "projects" && i+1 < len(pathSegments) {
			projectIdx = i + 1
			project = pathSegments[projectIdx]
			break
		}
	}

	if project == "" && queryParams != nil {
		project = queryParams.Get("project")
	}

	if project == "" {
		return "", "", "", errors.New("project ID not found in URL or path")
	}

	// Extract location
	locationIdx := -1
	for i, segment := range pathSegments {
		if segment == "locations" && i+1 < len(pathSegments) {
			locationIdx = i + 1
			location = pathSegments[locationIdx]
			break
		}
	}

	if location == "" {
		return "", "", "", errors.New("location not found in URL or path")
	}

	// Extract API ID
	apiIdx := -1
	for i, segment := range pathSegments {
		if segment == "apis" && i+1 < len(pathSegments) {
			apiIdx = i + 1
			apiID = pathSegments[apiIdx]
			break
		}
	}

	if apiID == "" {
		return "", "", "", errors.New("API ID not found in URL or path")
	}

	// Extract version ID
	versionIdx := -1
	for i, segment := range pathSegments {
		if segment == "versions" && i+1 < len(pathSegments) {
			versionIdx = i + 1
			versionID = pathSegments[versionIdx]
			break
		}
	}

	// Extract spec ID
	specIdx := -1
	for i, segment := range pathSegments {
		if segment == "specs" && i+1 < len(pathSegments) {
			specIdx = i + 1
			specID = pathSegments[specIdx]
			break
		}
	}

	apiResourceName := fmt.Sprintf("projects/%s/locations/%s/apis/%s", project, location, apiID)

	var apiVersionResourceName, apiSpecResourceName string
	if versionID != "" {
		apiVersionResourceName = fmt.Sprintf("%s/versions/%s", apiResourceName, versionID)
		if specID != "" {
			apiSpecResourceName = fmt.Sprintf("%s/specs/%s", apiVersionResourceName, specID)
		}
	}

	return apiResourceName, apiVersionResourceName, apiSpecResourceName, nil
}

// getAccessToken gets the access token for authentication.
func (c *APIHubClient) getAccessToken(ctx context.Context) (string, error) {
	if c.accessToken != "" {
		return c.accessToken, nil
	}

	if c.serviceAccount != "" {
		creds, err := google.CredentialsFromJSON(ctx, []byte(c.serviceAccount), "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return "", fmt.Errorf("failed to create credentials from JSON: %w", err)
		}
		token, err := creds.TokenSource.Token()
		if err != nil {
			return "", fmt.Errorf("failed to get token from credentials: %w", err)
		}
		return token.AccessToken, nil
	}

	// Try to use application default credentials
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("failed to find default credentials: %w", err)
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token from credentials: %w", err)
	}

	return token.AccessToken, nil
}
