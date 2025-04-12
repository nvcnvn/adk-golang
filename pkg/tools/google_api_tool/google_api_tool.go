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

package googleapitool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nvcnvn/adk-golang/pkg/auth"
	"github.com/nvcnvn/adk-golang/pkg/tools"
	"github.com/nvcnvn/adk-golang/pkg/tools/openapi_tool/openapi_spec_parser"
)

// GoogleApiTool wraps a RestApiTool to provide specialized functionality
// for Google API services.
type GoogleApiTool struct {
	restApiTool    *openapi_spec_parser.RestApiTool
	authCredential *auth.AuthCredential
}

// NewGoogleApiTool creates a new GoogleApiTool that wraps the provided RestApiTool.
func NewGoogleApiTool(restApiTool *openapi_spec_parser.RestApiTool) *GoogleApiTool {
	return &GoogleApiTool{
		restApiTool: restApiTool,
	}
}

// Name returns the name of the tool.
func (g *GoogleApiTool) Name() string {
	return g.restApiTool.Name()
}

// Description returns the description of the tool.
func (g *GoogleApiTool) Description() string {
	return g.restApiTool.Description()
}

// Execute implements the ExecutableTool interface for GoogleApiTool
func (g *GoogleApiTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	// We need to delegate to the underlying RestApiTool with the updated auth credentials
	// Since we can't directly update the RestApiTool's auth credential, we need to
	// possibly create a new instance with our auth credential when Execute is called

	// For now, simply try casting the Tool interface to an ExecutableTool
	if executableTool, ok := interface{}(g.restApiTool).(tools.ExecutableTool); ok {
		return executableTool.Execute(ctx, args)
	}

	return nil, fmt.Errorf("RestApiTool is not executable")
}

// ConfigureAuth configures OAuth 2.0 authentication for the tool.
func (g *GoogleApiTool) ConfigureAuth(clientID, clientSecret string) {
	// Store the auth credential
	g.authCredential = &auth.AuthCredential{
		AuthType: auth.OpenIDConnect,
		OAuth2: &auth.OAuth2Auth{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
	}
}
