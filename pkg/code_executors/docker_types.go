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

// Package code_executors provides functionality for executing code snippets.
package code_executors

import "io"

// Custom type definitions for Docker API
// Note: These are simplified versions of the Docker API types
// They are defined here to avoid issues with API version compatibility

// ExecConfig is the configuration for executing a command in a running container.
type ExecConfig struct {
	// User specifies the user name to use.
	User string
	// Privileged specifies whether the execution should be privileged.
	Privileged bool
	// Tty specifies whether a pseudo-TTY should be allocated.
	Tty bool
	// AttachStdin specifies whether to attach to stdin.
	AttachStdin bool
	// AttachStdout specifies whether to attach to stdout.
	AttachStdout bool
	// AttachStderr specifies whether to attach to stderr.
	AttachStderr bool
	// DetachKeys specifies the escape keys that detach the container.
	DetachKeys string
	// Env specifies the environment variables.
	Env []string
	// WorkingDir specifies the working directory.
	WorkingDir string
	// Cmd specifies the command to execute.
	Cmd []string
}

// ExecStartCheck is used to set up configuration for starting exec command.
type ExecStartCheck struct {
	// Detach specifies whether to detach from the command.
	Detach bool
	// Tty specifies whether a pseudo-TTY was allocated.
	Tty bool
}

// ContainerStartOptions holds options to start containers.
type ContainerStartOptions struct {
	// CheckpointID specifies the checkpoint to restore from.
	CheckpointID string
	// CheckpointDir specifies the directory where checkpoints are stored.
	CheckpointDir string
}

// ContainerRemoveOptions holds options to remove containers.
type ContainerRemoveOptions struct {
	// RemoveVolumes specifies whether to remove volumes attached to the container.
	RemoveVolumes bool
	// RemoveLinks specifies whether to remove links attached to the container.
	RemoveLinks bool
	// Force specifies whether to force removal of the container.
	Force bool
}

// HijackedResponse holds connection information for a hijacked request.
type HijackedResponse struct {
	// Conn is the underlying TCP connection.
	Conn io.Closer
	// Reader is a reader for the hijacked connection.
	Reader io.Reader
}
