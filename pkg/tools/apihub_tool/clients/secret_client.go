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

// Package clients provides client implementations for the apihub_tool.
package clients

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// SecretManagerClient is a client for Google Cloud Secret Manager.
type SecretManagerClient struct {
	client *secretmanager.Client
}

// SecretManagerClientOption represents options for creating a SecretManagerClient.
type SecretManagerClientOption struct {
	ServiceAccountJSON string
	AuthToken          string
}

// NewSecretManagerClient creates a new SecretManagerClient with the provided options.
func NewSecretManagerClient(ctx context.Context, opt SecretManagerClientOption) (*SecretManagerClient, error) {
	var clientOptions []option.ClientOption

	if opt.ServiceAccountJSON != "" {
		creds, err := google.CredentialsFromJSON(ctx, []byte(opt.ServiceAccountJSON), "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("failed to create credentials from JSON: %w", err)
		}
		clientOptions = append(clientOptions, option.WithCredentials(creds))
	} else if opt.AuthToken != "" {
		clientOptions = append(clientOptions, option.WithTokenSource(
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: opt.AuthToken})))
	} else {
		// Will use application default credentials
		creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}
		clientOptions = append(clientOptions, option.WithCredentials(creds))
	}

	client, err := secretmanager.NewClient(ctx, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret manager client: %w", err)
	}

	return &SecretManagerClient{
		client: client,
	}, nil
}

// GetSecret retrieves a secret from Google Cloud Secret Manager.
func (c *SecretManagerClient) GetSecret(ctx context.Context, resourceName string) (string, error) {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: resourceName,
	}

	result, err := c.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}

	return string(result.Payload.Data), nil
}

// Close closes the SecretManagerClient.
func (c *SecretManagerClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
