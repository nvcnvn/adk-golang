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

import (
	"encoding/json"
)

// AuthSchemeType defines the type of authentication scheme
type AuthSchemeType string

const (
	// APIKeyScheme - API Key scheme
	APIKeyScheme AuthSchemeType = "apiKey"

	// HTTPScheme - HTTP Auth scheme
	HTTPScheme AuthSchemeType = "http"

	// OAuth2Scheme - OAuth2 scheme
	OAuth2Scheme AuthSchemeType = "oauth2"

	// OpenIDConnectScheme - OpenID Connect scheme
	OpenIDConnectScheme AuthSchemeType = "openIdConnect"
)

// OAuthGrantType represents the OAuth2 flow (or grant type)
type OAuthGrantType string

const (
	// ClientCredentialsGrant - Client Credentials grant type
	ClientCredentialsGrant OAuthGrantType = "client_credentials"

	// AuthorizationCodeGrant - Authorization Code grant type
	AuthorizationCodeGrant OAuthGrantType = "authorization_code"

	// ImplicitGrant - Implicit grant type
	ImplicitGrant OAuthGrantType = "implicit"

	// PasswordGrant - Password grant type
	PasswordGrant OAuthGrantType = "password"
)

// OAuthFlow represents an OAuth2 flow configuration
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

// OAuthFlows represents OAuth2 flow configurations
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

// SecurityScheme represents a security scheme as defined in OpenAPI 3.0
type SecurityScheme struct {
	Type             AuthSchemeType         `json:"type"`
	Description      string                 `json:"description,omitempty"`
	Name             string                 `json:"name,omitempty"`             // for apiKey type
	In               string                 `json:"in,omitempty"`               // for apiKey type
	Scheme           string                 `json:"scheme,omitempty"`           // for http type
	BearerFormat     string                 `json:"bearerFormat,omitempty"`     // for http type with bearer scheme
	Flows            *OAuthFlows            `json:"flows,omitempty"`            // for oauth2 type
	OpenIDConnectURL string                 `json:"openIdConnectUrl,omitempty"` // for openIdConnect type
	ExtraFields      map[string]interface{} `json:"-"`
}

// UnmarshalJSON provides custom JSON unmarshaling with support for extra fields
func (s *SecurityScheme) UnmarshalJSON(data []byte) error {
	type Alias SecurityScheme
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	// First unmarshal the standard fields
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Then unmarshal everything to capture extra fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Remove the standard fields
	delete(raw, "type")
	delete(raw, "description")
	delete(raw, "name")
	delete(raw, "in")
	delete(raw, "scheme")
	delete(raw, "bearerFormat")
	delete(raw, "flows")
	delete(raw, "openIdConnectUrl")

	// Store the remaining fields
	s.ExtraFields = raw

	return nil
}

// MarshalJSON provides custom JSON marshaling with support for extra fields
func (s *SecurityScheme) MarshalJSON() ([]byte, error) {
	type Alias SecurityScheme
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	// Marshal the standard structure
	data, err := json.Marshal(aux)
	if err != nil {
		return nil, err
	}

	// If there are no extra fields, we're done
	if len(s.ExtraFields) == 0 {
		return data, nil
	}

	// Otherwise, we need to merge the extra fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Add extra fields
	for k, v := range s.ExtraFields {
		raw[k] = v
	}

	// Marshal the combined structure
	return json.Marshal(raw)
}

// Copy creates a deep copy of the SecurityScheme
func (s *SecurityScheme) Copy() *SecurityScheme {
	if s == nil {
		return nil
	}

	copy := &SecurityScheme{
		Type:             s.Type,
		Description:      s.Description,
		Name:             s.Name,
		In:               s.In,
		Scheme:           s.Scheme,
		BearerFormat:     s.BearerFormat,
		OpenIDConnectURL: s.OpenIDConnectURL,
	}

	// Deep copy flows if they exist
	if s.Flows != nil {
		copy.Flows = &OAuthFlows{}

		if s.Flows.Implicit != nil {
			copy.Flows.Implicit = &OAuthFlow{
				AuthorizationURL: s.Flows.Implicit.AuthorizationURL,
				TokenURL:         s.Flows.Implicit.TokenURL,
				RefreshURL:       s.Flows.Implicit.RefreshURL,
			}

			if s.Flows.Implicit.Scopes != nil {
				copy.Flows.Implicit.Scopes = make(map[string]string)
				for k, v := range s.Flows.Implicit.Scopes {
					copy.Flows.Implicit.Scopes[k] = v
				}
			}
		}

		if s.Flows.Password != nil {
			copy.Flows.Password = &OAuthFlow{
				AuthorizationURL: s.Flows.Password.AuthorizationURL,
				TokenURL:         s.Flows.Password.TokenURL,
				RefreshURL:       s.Flows.Password.RefreshURL,
			}

			if s.Flows.Password.Scopes != nil {
				copy.Flows.Password.Scopes = make(map[string]string)
				for k, v := range s.Flows.Password.Scopes {
					copy.Flows.Password.Scopes[k] = v
				}
			}
		}

		if s.Flows.ClientCredentials != nil {
			copy.Flows.ClientCredentials = &OAuthFlow{
				AuthorizationURL: s.Flows.ClientCredentials.AuthorizationURL,
				TokenURL:         s.Flows.ClientCredentials.TokenURL,
				RefreshURL:       s.Flows.ClientCredentials.RefreshURL,
			}

			if s.Flows.ClientCredentials.Scopes != nil {
				copy.Flows.ClientCredentials.Scopes = make(map[string]string)
				for k, v := range s.Flows.ClientCredentials.Scopes {
					copy.Flows.ClientCredentials.Scopes[k] = v
				}
			}
		}

		if s.Flows.AuthorizationCode != nil {
			copy.Flows.AuthorizationCode = &OAuthFlow{
				AuthorizationURL: s.Flows.AuthorizationCode.AuthorizationURL,
				TokenURL:         s.Flows.AuthorizationCode.TokenURL,
				RefreshURL:       s.Flows.AuthorizationCode.RefreshURL,
			}

			if s.Flows.AuthorizationCode.Scopes != nil {
				copy.Flows.AuthorizationCode.Scopes = make(map[string]string)
				for k, v := range s.Flows.AuthorizationCode.Scopes {
					copy.Flows.AuthorizationCode.Scopes[k] = v
				}
			}
		}
	}

	// Copy extra fields map
	if len(s.ExtraFields) > 0 {
		copy.ExtraFields = make(map[string]interface{})
		for k, v := range s.ExtraFields {
			copy.ExtraFields[k] = v
		}
	}

	return copy
}

// OpenIDConnectWithConfig represents an extended OpenID Connect security scheme with configuration
type OpenIDConnectWithConfig struct {
	Type                              AuthSchemeType         `json:"type"`
	AuthorizationEndpoint             string                 `json:"authorization_endpoint"`
	TokenEndpoint                     string                 `json:"token_endpoint"`
	UserinfoEndpoint                  string                 `json:"userinfo_endpoint,omitempty"`
	RevocationEndpoint                string                 `json:"revocation_endpoint,omitempty"`
	TokenEndpointAuthMethodsSupported []string               `json:"token_endpoint_auth_methods_supported,omitempty"`
	GrantTypesSupported               []string               `json:"grant_types_supported,omitempty"`
	Scopes                            []string               `json:"scopes,omitempty"`
	ExtraFields                       map[string]interface{} `json:"-"`
}

// UnmarshalJSON provides custom JSON unmarshaling with support for extra fields
func (o *OpenIDConnectWithConfig) UnmarshalJSON(data []byte) error {
	type Alias OpenIDConnectWithConfig
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(o),
	}

	// First unmarshal the standard fields
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Then unmarshal everything to capture extra fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Remove the standard fields
	delete(raw, "type")
	delete(raw, "authorization_endpoint")
	delete(raw, "token_endpoint")
	delete(raw, "userinfo_endpoint")
	delete(raw, "revocation_endpoint")
	delete(raw, "token_endpoint_auth_methods_supported")
	delete(raw, "grant_types_supported")
	delete(raw, "scopes")

	// Store the remaining fields
	o.ExtraFields = raw

	return nil
}

// MarshalJSON provides custom JSON marshaling with support for extra fields
func (o *OpenIDConnectWithConfig) MarshalJSON() ([]byte, error) {
	type Alias OpenIDConnectWithConfig
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(o),
	}

	// Marshal the standard structure
	data, err := json.Marshal(aux)
	if err != nil {
		return nil, err
	}

	// If there are no extra fields, we're done
	if len(o.ExtraFields) == 0 {
		return data, nil
	}

	// Otherwise, we need to merge the extra fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Add extra fields
	for k, v := range o.ExtraFields {
		raw[k] = v
	}

	// Marshal the combined structure
	return json.Marshal(raw)
}

// Copy creates a deep copy of the OpenIDConnectWithConfig
func (o *OpenIDConnectWithConfig) Copy() *OpenIDConnectWithConfig {
	if o == nil {
		return nil
	}

	copy := &OpenIDConnectWithConfig{
		Type:                  o.Type,
		AuthorizationEndpoint: o.AuthorizationEndpoint,
		TokenEndpoint:         o.TokenEndpoint,
		UserinfoEndpoint:      o.UserinfoEndpoint,
		RevocationEndpoint:    o.RevocationEndpoint,
	}

	// Deep copy slices
	if len(o.TokenEndpointAuthMethodsSupported) > 0 {
		copy.TokenEndpointAuthMethodsSupported = make([]string, len(o.TokenEndpointAuthMethodsSupported))
		for i, v := range o.TokenEndpointAuthMethodsSupported {
			copy.TokenEndpointAuthMethodsSupported[i] = v
		}
	}

	if len(o.GrantTypesSupported) > 0 {
		copy.GrantTypesSupported = make([]string, len(o.GrantTypesSupported))
		for i, v := range o.GrantTypesSupported {
			copy.GrantTypesSupported[i] = v
		}
	}

	if len(o.Scopes) > 0 {
		copy.Scopes = make([]string, len(o.Scopes))
		for i, v := range o.Scopes {
			copy.Scopes[i] = v
		}
	}

	// Copy extra fields map
	if len(o.ExtraFields) > 0 {
		copy.ExtraFields = make(map[string]interface{})
		for k, v := range o.ExtraFields {
			copy.ExtraFields[k] = v
		}
	}

	return copy
}

// AuthScheme represents either a SecurityScheme or OpenIDConnectWithConfig
type AuthScheme interface {
	// IsAuthScheme is a marker method to ensure type safety
	IsAuthScheme() bool
}

// Ensure SecurityScheme implements AuthScheme
func (s *SecurityScheme) IsAuthScheme() bool {
	return true
}

// Ensure OpenIDConnectWithConfig implements AuthScheme
func (o *OpenIDConnectWithConfig) IsAuthScheme() bool {
	return true
}

// GetAuthSchemeType returns the scheme type from an AuthScheme interface
func GetAuthSchemeType(scheme AuthScheme) AuthSchemeType {
	switch s := scheme.(type) {
	case *SecurityScheme:
		return s.Type
	case *OpenIDConnectWithConfig:
		return s.Type
	default:
		return ""
	}
}

// FromOAuthFlows determines the grant type from OAuthFlows
func FromOAuthFlows(flows *OAuthFlows) OAuthGrantType {
	if flows == nil {
		return ""
	}

	if flows.ClientCredentials != nil {
		return ClientCredentialsGrant
	}
	if flows.AuthorizationCode != nil {
		return AuthorizationCodeGrant
	}
	if flows.Implicit != nil {
		return ImplicitGrant
	}
	if flows.Password != nil {
		return PasswordGrant
	}
	return ""
}
