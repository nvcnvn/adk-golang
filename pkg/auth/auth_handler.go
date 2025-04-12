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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"net/url"
	"strings"
)

// SupportTokenExchange indicates whether token exchange is supported
// This would be set to true when an OAuth client library is available
var SupportTokenExchange = false

// Session interface represents a minimal interface for session state
type Session interface {
	// Get retrieves a value from session state
	Get(key string) (interface{}, bool)
	
	// Set stores a value in session state
	Set(key string, value interface{})
}

// AuthCredentialMissingError is returned when authentication credentials are missing
type AuthCredentialMissingError struct {
	Message string
}

func (e *AuthCredentialMissingError) Error() string {
	return e.Message
}

// AuthHandler handles authentication operations
type AuthHandler struct {
	AuthConfig AuthConfig
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(config AuthConfig) *AuthHandler {
	return &AuthHandler{
		AuthConfig: config,
	}
}

// ExchangeAuthToken generates an auth token from the authorization response
func (h *AuthHandler) ExchangeAuthToken() (*AuthCredential, error) {
	// Return the current token if token exchange isn't supported
	if !SupportTokenExchange {
		return h.AuthConfig.ExchangedAuthCredential, nil
	}
	
	var tokenEndpoint string
	var scopes []string
	
	// Extract token endpoint and scopes from different auth schemes
	switch scheme := h.AuthConfig.AuthScheme.(type) {
	case *OpenIDConnectWithConfig:
		if scheme.TokenEndpoint == "" {
			return h.AuthConfig.ExchangedAuthCredential, nil
		}
		tokenEndpoint = scheme.TokenEndpoint
		scopes = scheme.Scopes
	case *SecurityScheme:
		if scheme.Type != OAuth2Scheme || scheme.Flows == nil {
			return h.AuthConfig.ExchangedAuthCredential, nil
		}
		
		if scheme.Flows.AuthorizationCode == nil || scheme.Flows.AuthorizationCode.TokenURL == "" {
			return h.AuthConfig.ExchangedAuthCredential, nil
		}
		
		tokenEndpoint = scheme.Flows.AuthorizationCode.TokenURL
		
		if scheme.Flows.AuthorizationCode.Scopes != nil {
			for scope := range scheme.Flows.AuthorizationCode.Scopes {
				scopes = append(scopes, scope)
			}
		}
	default:
		return h.AuthConfig.ExchangedAuthCredential, nil
	}
	
	// Check if we have required OAuth2 fields
	cred := h.AuthConfig.ExchangedAuthCredential
	if cred == nil || 
	   cred.OAuth2 == nil || 
	   cred.OAuth2.ClientID == "" || 
	   cred.OAuth2.ClientSecret == "" ||
	   cred.OAuth2.Token != nil {
		return h.AuthConfig.ExchangedAuthCredential, nil
	}
	
	// In a real implementation, this would use an OAuth2 library to exchange tokens
	// For now, this is a placeholder where token exchange would happen
	
	// This is where we would call the OAuth2 client to exchange authorization code for tokens
	// Example pseudocode:
	// token, err := oauthClient.FetchToken(
	//     tokenEndpoint,
	//     authorizationResponse: cred.OAuth2.AuthResponseURI,
	//     code: cred.OAuth2.AuthCode,
	//     grantType: string(AuthorizationCodeGrant),
	// )
	
	// Create updated credential with token
	updatedCredential := &AuthCredential{
		AuthType: OAuth2,
		OAuth2: &OAuth2Auth{
			// In a real implementation, this would be set to the token from the OAuth2 client
			Token: map[string]interface{}{
				"access_token":  "placeholder_access_token",
				"refresh_token": "placeholder_refresh_token",
				"token_type":    "Bearer",
				"expires_in":    3600,
			},
		},
	}
	
	return updatedCredential, nil
}

// ParseAndStoreAuthResponse parses auth response and stores it in session
func (h *AuthHandler) ParseAndStoreAuthResponse(session Session) error {
	if session == nil {
		return errors.New("session is nil")
	}
	
	credentialKey := h.GetCredentialKey()
	
	// Store the current exchanged credential
	session.Set(credentialKey, h.AuthConfig.ExchangedAuthCredential)
	
	// Check if this is an OAuth2 or OpenID Connect scheme that needs token exchange
	schemeType := GetAuthSchemeType(h.AuthConfig.AuthScheme)
	if schemeType != OAuth2Scheme && schemeType != OpenIDConnectScheme {
		return nil
	}
	
	// Exchange token if needed
	exchangedToken, err := h.ExchangeAuthToken()
	if err != nil {
		return err
	}
	
	// Store the exchanged token
	session.Set(credentialKey, exchangedToken)
	
	return nil
}

// GetAuthResponse retrieves auth response from session
func (h *AuthHandler) GetAuthResponse(session Session) (*AuthCredential, error) {
	if session == nil {
		return nil, errors.New("session is nil")
	}
	
	credentialKey := h.GetCredentialKey()
	
	value, ok := session.Get(credentialKey)
	if !ok {
		return nil, nil
	}
	
	credential, ok := value.(*AuthCredential)
	if !ok {
		return nil, fmt.Errorf("stored value is not an AuthCredential")
	}
	
	return credential, nil
}

// GenerateAuthRequest generates an authentication request
func (h *AuthHandler) GenerateAuthRequest() (*AuthConfig, error) {
	// For schemes other than OAuth2 or OpenID Connect
	schemeType := GetAuthSchemeType(h.AuthConfig.AuthScheme)
	if schemeType != OAuth2Scheme && schemeType != OpenIDConnectScheme {
		return h.AuthConfig.Copy(), nil
	}
	
	// If auth_uri already exists in exchanged credential
	if h.AuthConfig.ExchangedAuthCredential != nil &&
	   h.AuthConfig.ExchangedAuthCredential.OAuth2 != nil &&
	   h.AuthConfig.ExchangedAuthCredential.OAuth2.AuthURI != "" {
		return h.AuthConfig.Copy(), nil
	}
	
	// Check if raw_auth_credential exists
	if h.AuthConfig.RawAuthCredential == nil {
		return nil, fmt.Errorf("auth scheme %s requires auth_credential", schemeType)
	}
	
	// Check if oauth2 exists in raw_auth_credential
	if h.AuthConfig.RawAuthCredential.OAuth2 == nil {
		return nil, fmt.Errorf("auth scheme %s requires oauth2 in auth_credential", schemeType)
	}
	
	// auth_uri in raw credential
	if h.AuthConfig.RawAuthCredential.OAuth2.AuthURI != "" {
		// Copy raw credential to exchanged credential
		config := h.AuthConfig.Copy()
		config.ExchangedAuthCredential = h.AuthConfig.RawAuthCredential.Copy()
		return config, nil
	}
	
	// Check for client_id and client_secret
	oauth2Cred := h.AuthConfig.RawAuthCredential.OAuth2
	if oauth2Cred.ClientID == "" || oauth2Cred.ClientSecret == "" {
		return nil, fmt.Errorf("auth scheme %s requires both client_id and client_secret in auth_credential.oauth2", schemeType)
	}
	
	// Generate new auth URI
	exchangedCredential, err := h.GenerateAuthURI()
	if err != nil {
		return nil, err
	}
	
	config := h.AuthConfig.Copy()
	config.ExchangedAuthCredential = exchangedCredential
	
	return config, nil
}

// GetCredentialKey generates a unique key for the given auth scheme and credential
func (h *AuthHandler) GetCredentialKey() string {
	var schemeName, credentialName string
	
	// Generate scheme name based on scheme type and hash of its JSON representation
	if h.AuthConfig.AuthScheme != nil {
		schemeType := GetAuthSchemeType(h.AuthConfig.AuthScheme)
		schemeHash := hashString(fmt.Sprintf("%v", h.AuthConfig.AuthScheme))
		schemeName = fmt.Sprintf("%s_%s", schemeType, schemeHash)
	}
	
	// Generate credential name based on credential type and hash of its JSON representation
	if h.AuthConfig.RawAuthCredential != nil {
		credType := h.AuthConfig.RawAuthCredential.AuthType
		credHash := hashString(fmt.Sprintf("%v", h.AuthConfig.RawAuthCredential))
		credentialName = fmt.Sprintf("%s_%s", credType, credHash)
	}
	
	return fmt.Sprintf("temp:adk_%s_%s", schemeName, credentialName)
}

// GenerateAuthURI generates an authorization URI
func (h *AuthHandler) GenerateAuthURI() (*AuthCredential, error) {
	var authorizationEndpoint string
	var scopes []string
	
	// Extract authorization endpoint and scopes from different auth schemes
	switch scheme := h.AuthConfig.AuthScheme.(type) {
	case *OpenIDConnectWithConfig:
		authorizationEndpoint = scheme.AuthorizationEndpoint
		scopes = scheme.Scopes
	case *SecurityScheme:
		if scheme.Flows == nil {
			return nil, errors.New("auth scheme has no flows configured")
		}
		
		if scheme.Flows.Implicit != nil && scheme.Flows.Implicit.AuthorizationURL != "" {
			authorizationEndpoint = scheme.Flows.Implicit.AuthorizationURL
			for scope := range scheme.Flows.Implicit.Scopes {
				scopes = append(scopes, scope)
			}
		} else if scheme.Flows.AuthorizationCode != nil && scheme.Flows.AuthorizationCode.AuthorizationURL != "" {
			authorizationEndpoint = scheme.Flows.AuthorizationCode.AuthorizationURL
			for scope := range scheme.Flows.AuthorizationCode.Scopes {
				scopes = append(scopes, scope)
			}
		} else if scheme.Flows.ClientCredentials != nil && scheme.Flows.ClientCredentials.TokenURL != "" {
			authorizationEndpoint = scheme.Flows.ClientCredentials.TokenURL
			for scope := range scheme.Flows.ClientCredentials.Scopes {
				scopes = append(scopes, scope)
			}
		} else if scheme.Flows.Password != nil && scheme.Flows.Password.TokenURL != "" {
			authorizationEndpoint = scheme.Flows.Password.TokenURL
			for scope := range scheme.Flows.Password.Scopes {
				scopes = append(scopes, scope)
			}
		} else {
			return nil, errors.New("no valid authorization URL found in auth scheme")
		}
	default:
		return nil, errors.New("invalid auth scheme type")
	}
	
	// Generate state for CSRF protection
	state, err := generateRandomString(16)
	if err != nil {
		return nil, err
	}
	
	// Build auth URI
	params := url.Values{}
	params.Add("client_id", h.AuthConfig.RawAuthCredential.OAuth2.ClientID)
	params.Add("response_type", "code")
	params.Add("state", state)
	params.Add("access_type", "offline")
	params.Add("prompt", "consent")
	
	if len(scopes) > 0 {
		params.Add("scope", strings.Join(scopes, " "))
	}
	
	if h.AuthConfig.RawAuthCredential.OAuth2.RedirectURI != "" {
		params.Add("redirect_uri", h.AuthConfig.RawAuthCredential.OAuth2.RedirectURI)
	}
	
	authURI := authorizationEndpoint
	if strings.Contains(authorizationEndpoint, "?") {
		authURI += "&" + params.Encode()
	} else {
		authURI += "?" + params.Encode()
	}
	
	// Create exchanged credential
	exchangedAuthCredential := h.AuthConfig.RawAuthCredential.Copy()
	if exchangedAuthCredential.OAuth2 == nil {
		exchangedAuthCredential.OAuth2 = &OAuth2Auth{}
	}
	exchangedAuthCredential.OAuth2.AuthURI = authURI
	exchangedAuthCredential.OAuth2.State = state
	
	return exchangedAuthCredential, nil
}

// hashString generates a simple hash string for a given input
func hashString(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum64())
}

// generateRandomString generates a random string of given length
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}