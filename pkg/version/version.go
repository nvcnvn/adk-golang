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

// Package version provides ADK version information.
package version

// Version is the current version of the ADK.
// This should be updated with each new release.
const Version = "0.1.0"

// UserAgent returns the user agent string for ADK.
func UserAgent() string {
	return "google-adk-golang/" + Version
}
