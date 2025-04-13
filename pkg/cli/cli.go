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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/nvcnvn/adk-golang/pkg/agents"
	"github.com/nvcnvn/adk-golang/pkg/runners"
	"github.com/nvcnvn/adk-golang/pkg/telemetry"
	"github.com/nvcnvn/adk-golang/pkg/version"
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
		Long:  "Run an interactive CLI for a specific agent.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentModule := args[0]
			saveSession, _ := cmd.Flags().GetBool("save_session")
			return runAgent(agentModule, saveSession)
		},
	}

	webCmd = &cobra.Command{
		Use:   "web [agents_dir]",
		Short: "Start the ADK web interface",
		Long:  "Start a web server with UI for agents.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentsDir := "."
			if len(args) > 0 {
				agentsDir = args[0]
			}
			sessionDBURL, _ := cmd.Flags().GetString("session_db_url")
			port, _ := cmd.Flags().GetInt("port")
			logToTmp, _ := cmd.Flags().GetBool("log_to_tmp")
			traceToCloud, _ := cmd.Flags().GetBool("trace_to_cloud")
			logLevel, _ := cmd.Flags().GetString("log_level")
			allowOrigins, _ := cmd.Flags().GetStringSlice("allow_origins")

			return startWebUI(agentsDir, sessionDBURL, port, logToTmp, traceToCloud, logLevel, allowOrigins)
		},
	}

	apiServerCmd = &cobra.Command{
		Use:   "api_server [agents_dir]",
		Short: "Start the ADK API server without UI",
		Long:  "Start a FastAPI server for agents without web UI.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentsDir := "."
			if len(args) > 0 {
				agentsDir = args[0]
			}
			sessionDBURL, _ := cmd.Flags().GetString("session_db_url")
			port, _ := cmd.Flags().GetInt("port")
			logToTmp, _ := cmd.Flags().GetBool("log_to_tmp")
			traceToCloud, _ := cmd.Flags().GetBool("trace_to_cloud")
			logLevel, _ := cmd.Flags().GetString("log_level")
			allowOrigins, _ := cmd.Flags().GetStringSlice("allow_origins")

			return startAPIServer(agentsDir, sessionDBURL, port, logToTmp, traceToCloud, logLevel, allowOrigins)
		},
	}

	evalCmd = &cobra.Command{
		Use:   "eval [agent_module_file_path] [eval_set_file_paths...]",
		Short: "Evaluate an agent with evaluation sets",
		Long:  "Evaluates an agent given the eval sets.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentPath := args[0]
			evalSetPaths := args[1:]
			configPath, _ := cmd.Flags().GetString("config_file_path")
			printDetailedResults, _ := cmd.Flags().GetBool("print_detailed_results")

			return evalAgent(agentPath, evalSetPaths, configPath, printDetailedResults)
		},
	}

	deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploy agent commands",
		Long:  "Commands for deploying agents to different environments.",
	}

	deployCloudRunCmd = &cobra.Command{
		Use:   "cloud_run [agent]",
		Short: "Deploy an agent to Cloud Run",
		Long:  "Deploys an agent to Google Cloud Run.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentPath := args[0]
			project, _ := cmd.Flags().GetString("project")
			region, _ := cmd.Flags().GetString("region")
			serviceName, _ := cmd.Flags().GetString("service_name")
			appName, _ := cmd.Flags().GetString("app_name")
			port, _ := cmd.Flags().GetInt("port")
			withCloudTrace, _ := cmd.Flags().GetBool("with_cloud_trace")
			withUI, _ := cmd.Flags().GetBool("with_ui")
			tempFolder, _ := cmd.Flags().GetString("temp_folder")

			return deployToCloudRun(agentPath, project, region, serviceName, appName, tempFolder, port, withCloudTrace, withUI)
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
	rootCmd.AddCommand(apiServerCmd)
	rootCmd.AddCommand(evalCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(versionCmd)

	// Add deploy subcommands
	deployCmd.AddCommand(deployCloudRunCmd)

	// Add flags for run command
	runCmd.Flags().BoolP("save_session", "", false, "Whether to save the session to a json file on exit")

	// Add flags for web command
	webCmd.Flags().StringP("session_db_url", "", "", "Database URL to store the session")
	webCmd.Flags().IntP("port", "", 8000, "Port for the web server")
	webCmd.Flags().StringSliceP("allow_origins", "", []string{}, "Additional origins to allow for CORS")
	webCmd.Flags().StringP("log_level", "", "INFO", "Set the logging level (DEBUG, INFO, WARNING, ERROR, CRITICAL)")
	webCmd.Flags().BoolP("log_to_tmp", "", false, "Whether to log to system temp folder instead of console")
	webCmd.Flags().BoolP("trace_to_cloud", "", false, "Whether to enable cloud trace for telemetry")

	// Add flags for api_server command (similar to web)
	apiServerCmd.Flags().StringP("session_db_url", "", "", "Database URL to store the session")
	apiServerCmd.Flags().IntP("port", "", 8000, "Port for the API server")
	apiServerCmd.Flags().StringSliceP("allow_origins", "", []string{}, "Additional origins to allow for CORS")
	apiServerCmd.Flags().StringP("log_level", "", "INFO", "Set the logging level (DEBUG, INFO, WARNING, ERROR, CRITICAL)")
	apiServerCmd.Flags().BoolP("log_to_tmp", "", false, "Whether to log to system temp folder instead of console")
	apiServerCmd.Flags().BoolP("trace_to_cloud", "", false, "Whether to enable cloud trace for telemetry")

	// Add flags for eval command
	evalCmd.Flags().StringP("config_file_path", "", "", "Path to config file")
	evalCmd.Flags().BoolP("print_detailed_results", "", false, "Whether to print detailed results on console")

	// Add flags for deploy cloud_run command
	deployCloudRunCmd.Flags().StringP("project", "", "", "Google Cloud project to deploy the agent")
	deployCloudRunCmd.Flags().StringP("region", "", "", "Google Cloud region to deploy the agent")
	deployCloudRunCmd.Flags().StringP("service_name", "", "adk-default-service-name", "Service name to use in Cloud Run")
	deployCloudRunCmd.Flags().StringP("app_name", "", "", "App name of the ADK API server")
	deployCloudRunCmd.Flags().IntP("port", "", 8000, "Port of the ADK API server")
	deployCloudRunCmd.Flags().BoolP("with_cloud_trace", "", false, "Whether to enable Cloud Trace for cloud run")
	deployCloudRunCmd.Flags().BoolP("with_ui", "", false, "Deploy ADK Web UI if set")
	deployCloudRunCmd.Flags().StringP("temp_folder", "", filepath.Join(os.TempDir(), "cloud_run_deploy_src", time.Now().Format("20060102_150405")), "Temp folder for the generated Cloud Run source files")

	// Add global flags
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	rootCmd.PersistentFlags().BoolVarP(&traceAll, "trace-all", "t", false, "Enable tracing for all operations")

	// Enable the simple tracer if debug mode is enabled
	if debug || traceAll {
		telemetry.SetDefaultTracer(telemetry.NewSimpleTracer())
	}
}

// runAgent loads and runs the specified agent module.
func runAgent(agentModule string, saveSession bool) error {
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

	// Configure the runner with save session option
	if saveSession {
		runner.SetSaveSessionEnabled(true)
	}

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

// startWebUI starts the web interface with UI.
func startWebUI(agentsDir, sessionDBURL string, port int, logToTmp, traceToCloud bool, logLevel string, allowOrigins []string) error {
	configureLogging(logToTmp, logLevel)

	successMessage := fmt.Sprintf(`
+-----------------------------------------------------------------------------+
| ADK Web Server started                                                      |
|                                                                             |
| For local testing, access at http://localhost:%-5d                          |
+-----------------------------------------------------------------------------+
`, port)

	color.Green(successMessage)

	// Start the web server
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	handler := getWebUIHandler(agentsDir, sessionDBURL, traceToCloud, allowOrigins)

	fmt.Println("Starting ADK web server...")
	return http.ListenAndServe(addr, handler)
}

// startAPIServer starts the API server without UI.
func startAPIServer(agentsDir, sessionDBURL string, port int, logToTmp, traceToCloud bool, logLevel string, allowOrigins []string) error {
	configureLogging(logToTmp, logLevel)

	// Start the API server
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	handler := getAPIHandler(agentsDir, sessionDBURL, traceToCloud, allowOrigins)

	fmt.Printf("Starting ADK API server on http://localhost:%d\n", port)
	return http.ListenAndServe(addr, handler)
}

// configureLogging sets up logging based on user preferences
func configureLogging(logToTmp bool, logLevel string) {
	if logToTmp {
		logFile, err := os.CreateTemp("", "adk-log-*.txt")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log file: %v\n", err)
			return
		}
		log.SetOutput(logFile)
		fmt.Printf("Logging to: %s\n", logFile.Name())
	} else {
		log.SetOutput(os.Stderr)
	}

	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	case "INFO":
		log.SetFlags(log.LstdFlags)
	case "WARNING", "ERROR", "CRITICAL":
		// In a real implementation, we'd configure log levels properly
		log.SetFlags(log.LstdFlags)
	}
}

// getWebUIHandler returns an HTTP handler for the web UI
func getWebUIHandler(agentsDir, sessionDBURL string, traceToCloud bool, allowOrigins []string) http.Handler {
	mux := http.NewServeMux()

	// Add API endpoints
	mux.Handle("/api/", getAPIHandler(agentsDir, sessionDBURL, traceToCloud, allowOrigins))

	// Add UI static files and routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintln(w, "<html><body><h1>ADK Web Interface</h1><p>This is a placeholder for the ADK Web UI.</p></body></html>")
			return
		}
		http.NotFound(w, r)
	})

	return mux
}

// getAPIHandler returns an HTTP handler for the API server
func getAPIHandler(agentsDir, sessionDBURL string, traceToCloud bool, allowOrigins []string) http.Handler {
	mux := http.NewServeMux()

	// Add API endpoints
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})

	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": version.Version})
	})

	// TODO: Add actual API handlers for agents, sessions, etc.

	return addCORS(mux, allowOrigins)
}

// addCORS adds CORS headers to the handler
func addCORS(h http.Handler, allowOrigins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		if len(allowOrigins) > 0 {
			origin := r.Header.Get("Origin")
			for _, allowed := range allowOrigins {
				if origin == allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// evalAgent evaluates an agent using the specified evaluation sets
func evalAgent(agentPath string, evalSetPaths []string, configPath string, printDetailedResults bool) error {
	fmt.Printf("Evaluating agent: %s\n", agentPath)
	fmt.Printf("Evaluation sets: %v\n", evalSetPaths)

	if configPath != "" {
		fmt.Printf("Using config file: %s\n", configPath)
		// TODO: Load configuration from the file
	}

	// Placeholder for evaluation implementation
	fmt.Println("Evaluation functionality is not yet implemented")

	// In a real implementation, we would:
	// 1. Load the agent
	// 2. Load the evaluation sets
	// 3. Run the evaluations
	// 4. Collect and display results

	return nil
}

// deployToCloudRun deploys an agent to Google Cloud Run
func deployToCloudRun(
	agentFolder, project, region, serviceName, appName, tempFolder string,
	port int, withCloudTrace, withUI bool) error {

	// Ensure the agent folder exists
	if _, err := os.Stat(agentFolder); os.IsNotExist(err) {
		return fmt.Errorf("agent folder %s does not exist", agentFolder)
	}

	// Create the temporary deployment folder
	if err := os.MkdirAll(tempFolder, 0755); err != nil {
		return fmt.Errorf("failed to create temp folder: %w", err)
	}

	fmt.Printf("Preparing Cloud Run deployment for agent: %s\n", agentFolder)
	fmt.Printf("Using temp folder: %s\n", tempFolder)

	// If app name is not provided, use the agent folder name
	if appName == "" {
		appName = filepath.Base(agentFolder)
	}

	// TODO: Implement the actual deployment:
	// 1. Copy the agent code to the temp folder
	// 2. Generate Dockerfile and other necessary files
	// 3. Build and push the Docker image
	// 4. Deploy to Cloud Run

	fmt.Println("Cloud Run deployment functionality is not yet implemented")

	return nil
}
