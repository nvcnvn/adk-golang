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

package auth

// AuthConfig represents the authentication configuration sent by a tool
// asking the client to collect auth credentials. ADK and the client will
// help fill in the response.
type AuthConfig struct {
	// AuthScheme is the auth scheme used to collect credentials
	AuthScheme AuthScheme `json:"auth_scheme"`

	// RawAuthCredential is the raw auth credential used to collect credentials.
	// It's used in some auth schemes that need to exchange auth credentials,
	// e.g., OAuth2 and OIDC. For other auth schemes, it could be nil.
	RawAuthCredential *AuthCredential `json:"raw_auth_credential,omitempty"`

	// ExchangedAuthCredential is the exchanged auth credential used to collect credentials.
	// ADK and client work together to fill it. For auth schemes that don't need to
	// exchange auth credentials, e.g., API key, service account, etc., it's filled by
	// client directly. For auth schemes that need to exchange auth credentials,
	// e.g., OAuth2 and OIDC, it's first filled by ADK. If the raw credentials
	// passed by a tool only has client ID and client credential, ADK will help to
	// generate the corresponding authorization URI and state and store the processed
	// credential in this field. If the raw credentials passed by a tool already has
	// authorization URI, state, etc., then it's copied to this field. Client will use
	// this field to guide the user through the OAuth2 flow and fill auth response in
	// this field.
	ExchangedAuthCredential *AuthCredential `json:"exchanged_auth_credential,omitempty"`
}

// Copy creates a deep copy of the AuthConfig
func (a *AuthConfig) Copy() *AuthConfig {
	if a == nil {
		return nil
	}

	copy := &AuthConfig{}

	// Copy AuthScheme
	switch scheme := a.AuthScheme.(type) {
	case *SecurityScheme:
		copy.AuthScheme = scheme.Copy()
	case *OpenIDConnectWithConfig:
		copy.AuthScheme = scheme.Copy()
	}

	// Copy credentials
	if a.RawAuthCredential != nil {
		copy.RawAuthCredential = a.RawAuthCredential.Copy()
	}

	if a.ExchangedAuthCredential != nil {
		copy.ExchangedAuthCredential = a.ExchangedAuthCredential.Copy()
	}

	return copy
}

// AuthToolArguments represents the arguments for the special long-running function tool
// that is used to request end-user credentials.
type AuthToolArguments struct {
	// FunctionCallID is the ID of the function call that requires authentication
	FunctionCallID string `json:"function_call_id"`

	// AuthConfig is the authentication configuration
	AuthConfig AuthConfig `json:"auth_config"`
}
