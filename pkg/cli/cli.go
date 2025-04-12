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

// Package cli provides the command line interface for the ADK.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/adk-golang/pkg/agents"
	"github.com/google/adk-golang/pkg/runners"
	"github.com/google/adk-golang/pkg/telemetry"
	"github.com/google/adk-golang/pkg/version"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:     "adk",
		Short:   "ADK - Agent Development Kit",
		Version: version.Version,
	}

	runCmd = &cobra.Command{
		Use:   "run [agent_module]",
		Short: "Run an agent in interactive mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentModule := args[0]
			return runAgent(agentModule)
		},
	}

	webCmd = &cobra.Command{
		Use:   "web",
		Short: "Start the ADK web interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			return startWebUI()
		},
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the ADK version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ADK version: %s\n", version.Version)
		},
	}

	// Flags
	port     int
	debug    bool
	traceAll bool
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add commands to root
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(webCmd)
	rootCmd.AddCommand(versionCmd)

	// Add flags
	webCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for the web UI")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	rootCmd.PersistentFlags().BoolVarP(&traceAll, "trace-all", "t", false, "Enable tracing for all operations")

	// Enable the simple tracer if debug mode is enabled
	if debug || traceAll {
		telemetry.SetDefaultTracer(telemetry.NewSimpleTracer())
	}
}

// runAgent loads and runs the specified agent module.
func runAgent(agentModule string) error {
	fmt.Printf("Loading agent from module: %s\n", agentModule)

	// If the agent module is a file path, load the agent from the file
	absPath, err := getAgentPath(agentModule)
	if err != nil {
		return err
	}

	// Build and run the agent module
	if err := buildAndRunAgentModule(absPath); err != nil {
		return err
	}

	// Get the exported agent
	agent, ok := agents.GetExportedAgent(filepath.Base(agentModule))
	if !ok {
		return fmt.Errorf("agent %s not exported", agentModule)
	}

	fmt.Printf("Using model: %s\n", agent.Model())
	fmt.Printf("Agent description: %s\n", agent.Description())

	// Create a runner for the agent
	runner := runners.NewSimpleRunner()
	ctx := context.Background()

	// Run the agent in interactive mode
	return runner.RunInteractive(ctx, agent, os.Stdin, os.Stdout)
}

// getAgentPath gets the absolute path to the agent module.
func getAgentPath(agentModule string) (string, error) {
	// If it's an absolute path, return it directly
	if filepath.IsAbs(agentModule) {
		return agentModule, nil
	}

	// If it's a relative path, make it absolute
	absPath, err := filepath.Abs(agentModule)
	if err != nil {
		return "", err
	}

	// If it's a directory, look for main.go in that directory
	fileInfo, err := os.Stat(absPath)
	if err == nil && fileInfo.IsDir() {
		mainFile := filepath.Join(absPath, "main.go")
		if _, err := os.Stat(mainFile); err == nil {
			return mainFile, nil
		}
		return "", fmt.Errorf("no main.go found in %s", absPath)
	}

	// Otherwise, return the absolute path
	return absPath, nil
}

// buildAndRunAgentModule builds and runs the agent module to register the agent.
func buildAndRunAgentModule(agentPath string) error {
	if strings.HasSuffix(agentPath, ".go") {
		// Compile and run the Go file
		dir := filepath.Dir(agentPath)
		fileName := filepath.Base(agentPath)

		// Build the Go file
		buildCmd := exec.Command("go", "build", "-o", "agent_temp", fileName)
		buildCmd.Dir = dir
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("failed to build agent module: %w", err)
		}

		// Run the compiled binary
		runCmd := exec.Command("./agent_temp")
		runCmd.Dir = dir
		runCmd.Stdout = os.Stdout
		runCmd.Stderr = os.Stderr
		if err := runCmd.Run(); err != nil {
			return fmt.Errorf("failed to run agent module: %w", err)
		}

		// Clean up
		os.Remove(filepath.Join(dir, "agent_temp"))
	} else {
		return errors.New("unsupported agent module format")
	}

	return nil
}

// startWebUI starts the web interface.
func startWebUI() error {
	fmt.Printf("Starting ADK Web UI on http://localhost:%d\n", port)
	fmt.Println("(This is a placeholder. The actual web UI implementation needs to be completed.)")

	// In an actual implementation, this would start a web server
	// For now, we'll just keep the process running until interrupted
	fmt.Println("Press Ctrl+C to stop the web UI")
	<-make(chan struct{})

	return nil
}
