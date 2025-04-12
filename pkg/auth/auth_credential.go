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

// Package auth provides authentication functionality for the ADK.
package auth

import (
	"encoding/json"
)

// AuthCredentialType represents the type of authentication credential.
type AuthCredentialType string

const (
	// APIKey - API Key credential:
	// https://swagger.io/docs/specification/v3_0/authentication/api-keys/
	APIKey AuthCredentialType = "apiKey"

	// HTTP - Credentials for HTTP Auth schemes:
	// https://www.iana.org/assignments/http-authschemes/http-authschemes.xhtml
	HTTP AuthCredentialType = "http"

	// OAuth2 - OAuth2 credentials:
	// https://swagger.io/docs/specification/v3_0/authentication/oauth2/
	OAuth2 AuthCredentialType = "oauth2"

	// OpenIDConnect - OpenID Connect credentials:
	// https://swagger.io/docs/specification/v3_0/authentication/openid-connect-discovery/
	OpenIDConnect AuthCredentialType = "openIdConnect"

	// ServiceAccount - Service Account credentials:
	// https://cloud.google.com/iam/docs/service-account-creds
	ServiceAccountType AuthCredentialType = "serviceAccount"
)

// HttpCredentials represents the secret token value for HTTP authentication.
// This includes user name, password, oauth token, etc.
type HttpCredentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// HttpAuth represents the credentials and metadata for HTTP authentication.
type HttpAuth struct {
	// Scheme is the name of the HTTP Authorization scheme to be used in the Authorization
	// header as defined in RFC7235. The values used SHOULD be registered in the
	// IANA Authentication Scheme registry.
	// Examples: 'basic', 'bearer'
	Scheme      string          `json:"scheme"`
	Credentials HttpCredentials `json:"credentials"`
}

// OAuth2Auth represents credential value and its metadata for a OAuth2 credential.
type OAuth2Auth struct {
	ClientID        string                 `json:"client_id,omitempty"`
	ClientSecret    string                 `json:"client_secret,omitempty"`
	AuthURI         string                 `json:"auth_uri,omitempty"`
	State           string                 `json:"state,omitempty"`
	RedirectURI     string                 `json:"redirect_uri,omitempty"`
	AuthResponseURI string                 `json:"auth_response_uri,omitempty"`
	AuthCode        string                 `json:"auth_code,omitempty"`
	Token           map[string]interface{} `json:"token,omitempty"`
}

// ServiceAccountCredential represents Google Service Account configuration.
type ServiceAccountCredential struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
	UniverseDomain          string `json:"universe_domain"`
}

// ServiceAccountAuth represents Google Service Account configuration.
type ServiceAccountAuth struct {
	ServiceAccountCredential *ServiceAccountCredential `json:"service_account_credential,omitempty"`
	Scopes                   []string                  `json:"scopes"`
	UseDefaultCredential     bool                      `json:"use_default_credential,omitempty"`
}

// AuthCredential represents an authentication credential.
type AuthCredential struct {
	AuthType    AuthCredentialType     `json:"auth_type"`
	ResourceRef string                 `json:"resource_ref,omitempty"`
	APIKey      string                 `json:"api_key,omitempty"`
	HTTP        *HttpAuth              `json:"http,omitempty"`
	ServiceAcct *ServiceAccountAuth    `json:"service_account,omitempty"`
	OAuth2      *OAuth2Auth            `json:"oauth2,omitempty"`
	ExtraFields map[string]interface{} `json:"-"`
}

// UnmarshalJSON provides custom JSON unmarshaling with support for extra fields
func (a *AuthCredential) UnmarshalJSON(data []byte) error {
	type Alias AuthCredential
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(a),
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
	delete(raw, "auth_type")
	delete(raw, "resource_ref")
	delete(raw, "api_key")
	delete(raw, "http")
	delete(raw, "service_account")
	delete(raw, "oauth2")

	// Store the remaining fields
	a.ExtraFields = raw

	return nil
}

// MarshalJSON provides custom JSON marshaling with support for extra fields
func (a *AuthCredential) MarshalJSON() ([]byte, error) {
	type Alias AuthCredential
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	// Marshal the standard structure
	data, err := json.Marshal(aux)
	if err != nil {
		return nil, err
	}

	// If there are no extra fields, we're done
	if len(a.ExtraFields) == 0 {
		return data, nil
	}

	// Otherwise, we need to merge the extra fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Add extra fields
	for k, v := range a.ExtraFields {
		raw[k] = v
	}

	// Marshal the combined structure
	return json.Marshal(raw)
}

// Copy creates a deep copy of the AuthCredential
func (a *AuthCredential) Copy() *AuthCredential {
	if a == nil {
		return nil
	}

	result := &AuthCredential{
		AuthType:    a.AuthType,
		ResourceRef: a.ResourceRef,
		APIKey:      a.APIKey,
	}

	if a.HTTP != nil {
		result.HTTP = &HttpAuth{
			Scheme: a.HTTP.Scheme,
			Credentials: HttpCredentials{
				Username: a.HTTP.Credentials.Username,
				Password: a.HTTP.Credentials.Password,
				Token:    a.HTTP.Credentials.Token,
			},
		}
	}

	if a.OAuth2 != nil {
		oauth2Copy := &OAuth2Auth{
			ClientID:        a.OAuth2.ClientID,
			ClientSecret:    a.OAuth2.ClientSecret,
			AuthURI:         a.OAuth2.AuthURI,
			State:           a.OAuth2.State,
			RedirectURI:     a.OAuth2.RedirectURI,
			AuthResponseURI: a.OAuth2.AuthResponseURI,
			AuthCode:        a.OAuth2.AuthCode,
		}

		if a.OAuth2.Token != nil {
			tokenCopy := make(map[string]interface{})
			for k, v := range a.OAuth2.Token {
				tokenCopy[k] = v
			}
			oauth2Copy.Token = tokenCopy
		}

		result.OAuth2 = oauth2Copy
	}

	if a.ServiceAcct != nil {
		serviceCopy := &ServiceAccountAuth{
			UseDefaultCredential: a.ServiceAcct.UseDefaultCredential,
			Scopes:               make([]string, len(a.ServiceAcct.Scopes)),
		}

		// Copy slices
		copy(serviceCopy.Scopes, a.ServiceAcct.Scopes)

		// Copy service account credential if it exists
		if a.ServiceAcct.ServiceAccountCredential != nil {
			serviceCopy.ServiceAccountCredential = &ServiceAccountCredential{
				Type:                    a.ServiceAcct.ServiceAccountCredential.Type,
				ProjectID:               a.ServiceAcct.ServiceAccountCredential.ProjectID,
				PrivateKeyID:            a.ServiceAcct.ServiceAccountCredential.PrivateKeyID,
				PrivateKey:              a.ServiceAcct.ServiceAccountCredential.PrivateKey,
				ClientEmail:             a.ServiceAcct.ServiceAccountCredential.ClientEmail,
				ClientID:                a.ServiceAcct.ServiceAccountCredential.ClientID,
				AuthURI:                 a.ServiceAcct.ServiceAccountCredential.AuthURI,
				TokenURI:                a.ServiceAcct.ServiceAccountCredential.TokenURI,
				AuthProviderX509CertURL: a.ServiceAcct.ServiceAccountCredential.AuthProviderX509CertURL,
				ClientX509CertURL:       a.ServiceAcct.ServiceAccountCredential.ClientX509CertURL,
				UniverseDomain:          a.ServiceAcct.ServiceAccountCredential.UniverseDomain,
			}
		}

		result.ServiceAcct = serviceCopy
	}

	// Copy extra fields map
	if len(a.ExtraFields) > 0 {
		result.ExtraFields = make(map[string]interface{})
		for k, v := range a.ExtraFields {
			result.ExtraFields[k] = v
		}
	}

	return result
}
