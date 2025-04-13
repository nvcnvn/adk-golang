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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

// Session represents a minimal interface for session state
type Session interface {
	// Get retrieves a value from session state
	Get(key string) (interface{}, bool)

	// Set stores a value in session state
	Set(key string, value interface{})
}

// AuthHandler handles authentication operations
type AuthHandler struct {
	Config AuthConfig
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(config AuthConfig) *AuthHandler {
	return &AuthHandler{
		Config: config,
	}
}

// ExchangeAuthToken generates an auth token from the authorization response
func (h *AuthHandler) ExchangeAuthToken() (*AuthCredential, error) {
	authScheme := h.Config.AuthScheme
	authCred := h.Config.ExchangedAuthCredential

	if authCred == nil || authCred.OAuth2 == nil || authCred.OAuth2.Token != nil {
		return authCred, nil
	}

	var tokenEndpoint string
	var scopes []string

	switch scheme := authScheme.(type) {
	case *OpenIDConnectWithConfig:
		if scheme.TokenEndpoint == "" {
			return authCred, nil
		}
		tokenEndpoint = scheme.TokenEndpoint
		scopes = scheme.Scopes
	case *SecurityScheme:
		if scheme.Type != OAuth2Scheme || scheme.Flows == nil ||
			scheme.Flows.AuthorizationCode == nil ||
			scheme.Flows.AuthorizationCode.TokenURL == "" {
			return authCred, nil
		}
		tokenEndpoint = scheme.Flows.AuthorizationCode.TokenURL

		// Extract scopes from map keys
		if scheme.Flows.AuthorizationCode.Scopes != nil {
			scopes = make([]string, 0, len(scheme.Flows.AuthorizationCode.Scopes))
			for scope := range scheme.Flows.AuthorizationCode.Scopes {
				scopes = append(scopes, scope)
			}
		}
	default:
		return authCred, nil
	}

	if authCred.OAuth2.ClientID == "" || authCred.OAuth2.ClientSecret == "" {
		return authCred, errors.New("client ID and secret are required for token exchange")
	}

	// Create OAuth2 config
	oauth2Config := &oauth2.Config{
		ClientID:     authCred.OAuth2.ClientID,
		ClientSecret: authCred.OAuth2.ClientSecret,
		RedirectURL:  authCred.OAuth2.RedirectURI,
		Endpoint: oauth2.Endpoint{
			TokenURL: tokenEndpoint,
		},
		Scopes: scopes,
	}

	// Exchange authorization code for token
	token, err := oauth2Config.Exchange(
		oauth2.NoContext,
		authCred.OAuth2.AuthCode,
		oauth2.SetAuthURLParam("grant_type", string(AuthorizationCodeGrant)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %v", err)
	}

	// Convert token to map
	tokenMap := make(map[string]interface{})
	tokenMap["access_token"] = token.AccessToken
	tokenMap["token_type"] = token.TokenType
	tokenMap["refresh_token"] = token.RefreshToken
	tokenMap["expiry"] = token.Expiry

	// Create a new credential with the token
	updatedCred := &AuthCredential{
		AuthType: OAuth2,
		OAuth2: &OAuth2Auth{
			Token: tokenMap,
		},
	}

	return updatedCred, nil
}

// ParseAndStoreAuthResponse parses auth response and stores it in session
func (h *AuthHandler) ParseAndStoreAuthResponse(session Session) error {
	credKey := h.GetCredentialKey()

	session.Set(credKey, h.Config.ExchangedAuthCredential)

	authSchemeType := GetAuthSchemeType(h.Config.AuthScheme)
	if authSchemeType != OAuth2Scheme && authSchemeType != OpenIDConnectScheme {
		return nil
	}

	// Exchange token if needed
	exchangedCred, err := h.ExchangeAuthToken()
	if err != nil {
		return err
	}

	session.Set(credKey, exchangedCred)
	return nil
}

// GetAuthResponse retrieves auth response from session
func (h *AuthHandler) GetAuthResponse(session Session) *AuthCredential {
	credKey := h.GetCredentialKey()
	val, ok := session.Get(credKey)
	if !ok {
		return nil
	}

	cred, ok := val.(*AuthCredential)
	if !ok {
		return nil
	}

	return cred
}

// GenerateAuthRequest generates auth request
func (h *AuthHandler) GenerateAuthRequest() (*AuthConfig, error) {
	authSchemeType := GetAuthSchemeType(h.Config.AuthScheme)
	if authSchemeType != OAuth2Scheme && authSchemeType != OpenIDConnectScheme {
		return h.Config.Copy(), nil
	}

	// Auth URI already in exchanged credential
	if h.Config.ExchangedAuthCredential != nil &&
		h.Config.ExchangedAuthCredential.OAuth2 != nil &&
		h.Config.ExchangedAuthCredential.OAuth2.AuthURI != "" {
		return h.Config.Copy(), nil
	}

	// Check if raw_auth_credential exists
	if h.Config.RawAuthCredential == nil {
		return nil, fmt.Errorf("auth scheme %s requires auth_credential", authSchemeType)
	}

	// Check if oauth2 exists in raw_auth_credential
	if h.Config.RawAuthCredential.OAuth2 == nil {
		return nil, fmt.Errorf("auth scheme %s requires oauth2 in auth_credential", authSchemeType)
	}

	// Auth URI in raw credential
	if h.Config.RawAuthCredential.OAuth2.AuthURI != "" {
		return &AuthConfig{
			AuthScheme:              h.Config.AuthScheme,
			RawAuthCredential:       h.Config.RawAuthCredential,
			ExchangedAuthCredential: h.Config.RawAuthCredential.Copy(),
		}, nil
	}

	// Check for client_id and client_secret
	if h.Config.RawAuthCredential.OAuth2.ClientID == "" || h.Config.RawAuthCredential.OAuth2.ClientSecret == "" {
		return nil, fmt.Errorf("auth scheme %s requires both client_id and client_secret in auth_credential.oauth2", authSchemeType)
	}

	// Generate new auth URI
	exchangedCred, err := h.GenerateAuthURI()
	if err != nil {
		return nil, err
	}

	return &AuthConfig{
		AuthScheme:              h.Config.AuthScheme,
		RawAuthCredential:       h.Config.RawAuthCredential,
		ExchangedAuthCredential: exchangedCred,
	}, nil
}

// GetCredentialKey generates a unique key for the given auth scheme and credential
func (h *AuthHandler) GetCredentialKey() string {
	authScheme := h.Config.AuthScheme
	authCred := h.Config.RawAuthCredential

	var schemeName string
	if authScheme != nil {
		schemeType := GetAuthSchemeType(authScheme)
		schemeJSON, _ := json.Marshal(authScheme)
		hash := sha256.Sum256(schemeJSON)
		schemeName = fmt.Sprintf("%s_%x", schemeType, hash[:4])
	}

	var credName string
	if authCred != nil {
		credJSON, _ := json.Marshal(authCred)
		hash := sha256.Sum256(credJSON)
		credName = fmt.Sprintf("%s_%x", authCred.AuthType, hash[:4])
	}

	return fmt.Sprintf("temp:adk_%s_%s", schemeName, credName)
}

// GenerateAuthURI generates an auth URI for OAuth2 authorization
func (h *AuthHandler) GenerateAuthURI() (*AuthCredential, error) {
	authScheme := h.Config.AuthScheme
	authCred := h.Config.RawAuthCredential

	var authEndpoint string
	var scopes []string

	switch scheme := authScheme.(type) {
	case *OpenIDConnectWithConfig:
		authEndpoint = scheme.AuthorizationEndpoint
		scopes = scheme.Scopes
	case *SecurityScheme:
		if scheme.Flows == nil {
			return nil, errors.New("oauth flows not defined in security scheme")
		}

		// Get auth URL based on available flows
		if scheme.Flows.Implicit != nil && scheme.Flows.Implicit.AuthorizationURL != "" {
			authEndpoint = scheme.Flows.Implicit.AuthorizationURL
			if scheme.Flows.Implicit.Scopes != nil {
				scopes = make([]string, 0, len(scheme.Flows.Implicit.Scopes))
				for scope := range scheme.Flows.Implicit.Scopes {
					scopes = append(scopes, scope)
				}
			}
		} else if scheme.Flows.AuthorizationCode != nil && scheme.Flows.AuthorizationCode.AuthorizationURL != "" {
			authEndpoint = scheme.Flows.AuthorizationCode.AuthorizationURL
			if scheme.Flows.AuthorizationCode.Scopes != nil {
				scopes = make([]string, 0, len(scheme.Flows.AuthorizationCode.Scopes))
				for scope := range scheme.Flows.AuthorizationCode.Scopes {
					scopes = append(scopes, scope)
				}
			}
		} else if scheme.Flows.ClientCredentials != nil && scheme.Flows.ClientCredentials.TokenURL != "" {
			authEndpoint = scheme.Flows.ClientCredentials.TokenURL
			if scheme.Flows.ClientCredentials.Scopes != nil {
				scopes = make([]string, 0, len(scheme.Flows.ClientCredentials.Scopes))
				for scope := range scheme.Flows.ClientCredentials.Scopes {
					scopes = append(scopes, scope)
				}
			}
		} else if scheme.Flows.Password != nil && scheme.Flows.Password.TokenURL != "" {
			authEndpoint = scheme.Flows.Password.TokenURL
			if scheme.Flows.Password.Scopes != nil {
				scopes = make([]string, 0, len(scheme.Flows.Password.Scopes))
				for scope := range scheme.Flows.Password.Scopes {
					scopes = append(scopes, scope)
				}
			}
		} else {
			return nil, errors.New("no valid authorization URL found in security scheme")
		}
	default:
		return nil, errors.New("unsupported auth scheme type")
	}

	// Generate a unique state
	stateBytes := []byte(fmt.Sprintf("%s_%s_%d", authCred.OAuth2.ClientID, authCred.OAuth2.RedirectURI, stringHash(scopes)))
	state := fmt.Sprintf("%x", sha256.Sum256(stateBytes))

	// Build auth URL
	values := url.Values{}
	values.Set("client_id", authCred.OAuth2.ClientID)
	values.Set("redirect_uri", authCred.OAuth2.RedirectURI)
	values.Set("response_type", "code")
	values.Set("state", state)
	values.Set("access_type", "offline")
	values.Set("prompt", "consent")
	if len(scopes) > 0 {
		values.Set("scope", strings.Join(scopes, " "))
	}

	authURL := fmt.Sprintf("%s?%s", authEndpoint, values.Encode())

	// Create a copy of the credential with the auth URI
	exchangedAuthCred := authCred.Copy()
	exchangedAuthCred.OAuth2.AuthURI = authURL
	exchangedAuthCred.OAuth2.State = state

	return exchangedAuthCred, nil
}

// Helper function to create a hash of a string slice
func stringHash(strs []string) int {
	hash := 0
	for _, s := range strs {
		for i, c := range s {
			hash += i * int(c)
		}
	}
	return hash
}
