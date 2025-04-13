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
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// walkToRootUntilFound walks up the directory tree starting from the given folder
// searching for a file with the given filename. Returns the full path of the file
// if found, or an empty string otherwise.
func walkToRootUntilFound(folder, filename string) string {
	checkPath := filepath.Join(folder, filename)
	if fileInfo, err := os.Stat(checkPath); err == nil && !fileInfo.IsDir() {
		return checkPath
	}

	parentFolder := filepath.Dir(folder)
	// Check if we've reached the root
	if parentFolder == folder {
		return ""
	}

	return walkToRootUntilFound(parentFolder, filename)
}

// LoadDotEnvForAgent loads the .env file for the agent module.
// It starts looking for the .env file in the agent's directory and walks up
// to the root directory until it finds the file.
func LoadDotEnvForAgent(agentName string, agentParentFolder string, filename string) {
	if filename == "" {
		filename = ".env"
	}

	// Get the folder of the agent module as starting folder
	startingFolder := filepath.Join(agentParentFolder, agentName)
	startingFolder, err := filepath.Abs(startingFolder)
	if err != nil {
		log.Printf("Error getting absolute path for agent folder: %v", err)
		return
	}

	dotenvFilePath := walkToRootUntilFound(startingFolder, filename)
	if dotenvFilePath != "" {
		err := godotenv.Load(dotenvFilePath)
		if err != nil {
			log.Printf("Error loading %s file for %s: %v", filename, agentName, err)
			return
		}
		log.Printf("Loaded %s file for %s at %s", filename, agentName, dotenvFilePath)
	} else {
		log.Printf("No %s file found for %s", filename, agentName)
	}
}
