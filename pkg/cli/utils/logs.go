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

package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// LogConfig holds configuration for logging
type LogConfig struct {
	// LogLevel represents the logging level (Debug = 0, Info = 1, Warning = 2, Error = 3)
	LogLevel int
	// SubFolder is the subfolder name in the temp directory to store logs
	SubFolder string
	// LogFilePrefix is the prefix for the log file name
	LogFilePrefix string
	// LogFileTimestamp is the timestamp format for log file names
	LogFileTimestamp string
}

// DefaultLogConfig returns a default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		LogLevel:         1, // Info level
		SubFolder:        "agents_log",
		LogFilePrefix:    "agent",
		LogFileTimestamp: time.Now().Format("20060102_150405"),
	}
}

// LogToStderr configures logging to the standard error output
func LogToStderr(logLevel int) {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// LogToTmpFolder configures logging to a file in the system temp directory
// Returns the path to the log file
func LogToTmpFolder(config *LogConfig) (string, error) {
	if config == nil {
		config = DefaultLogConfig()
	}

	// Create log directory
	logDir := filepath.Join(os.TempDir(), config.SubFolder)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file
	logFilename := fmt.Sprintf("%s.%s.log", config.LogFilePrefix, config.LogFileTimestamp)
	logFilepath := filepath.Join(logDir, logFilename)

	file, err := os.Create(logFilepath)
	if err != nil {
		return "", fmt.Errorf("failed to create log file: %w", err)
	}

	// Set up logging to file
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Create symlink to latest log file
	latestLogLink := filepath.Join(logDir, fmt.Sprintf("%s.latest.log", config.LogFilePrefix))

	// Remove existing symlink if it exists
	os.Remove(latestLogLink) // Ignore errors if file doesn't exist

	// Create new symlink
	if err := os.Symlink(logFilepath, latestLogLink); err != nil {
		// Continue even if symlink creation fails, just log the error
		fmt.Fprintf(os.Stderr, "Warning: Failed to create symlink to log file: %v\n", err)
	} else {
		fmt.Printf("To access latest log: tail -F %s\n", latestLogLink)
	}

	fmt.Printf("Log setup complete: %s\n", logFilepath)
	return logFilepath, nil
}

// CreateMultiLogger creates a logger that writes to both a file and stderr
func CreateMultiLogger(config *LogConfig) (string, error) {
	if config == nil {
		config = DefaultLogConfig()
	}

	// Get file writer
	logFilepath, err := LogToTmpFolder(config)
	if err != nil {
		return "", err
	}

	// Open the file for reading
	file, err := os.OpenFile(logFilepath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open log file: %w", err)
	}

	// Create a multi-writer that writes to both stderr and the file
	multiWriter := io.MultiWriter(os.Stderr, file)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return logFilepath, nil
}
