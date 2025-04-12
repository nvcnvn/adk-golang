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

// ConnectionDetails contains information about a connection
type ConnectionDetails struct {
	// ServiceName is the service directory name for the connection
	ServiceName string
	// Host is the host name in case of TLS service directory
	Host string
	// AuthOverrideEnabled indicates if auth override is enabled for the connection
	AuthOverrideEnabled bool
}

// ActionDetails contains details about an action
type ActionDetails struct {
	// InputSchema is the JSON schema for the action's input
	InputSchema map[string]interface{}
	// OutputSchema is the JSON schema for the action's output
	OutputSchema map[string]interface{}
	// Description of the action
	Description string
	// DisplayName is the display name of the action
	DisplayName string
}

// LongRunningOperation represents a Google Cloud long-running operation
type LongRunningOperation struct {
	// Name is the operation ID
	Name string
	// Done indicates if the operation is complete
	Done bool
	// Response contains the operation's response
	Response map[string]interface{}
}
