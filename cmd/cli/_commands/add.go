// Copyright 2025 Matteo Brambilla - TEADAL
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

package commands

import (
	"dspn-regogenerator/internal/usecases"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	openAPISpec string
)

func loadSpecFile(specFile string) ([]byte, error) {
	// Load the OpenAPI spec file
	specData, err := os.ReadFile(specFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI spec file: %v", err)
	}
	return specData, nil
}

var AddCmd = &cobra.Command{
	Use:   "add [--spec <path/to/openapi/spec>] <service name>",
	Short: "Add policies related to a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]
		if openAPISpec == "" {
			cmd.Help()
			return
		}

		// Load the OpenAPI spec file
		specData, err := loadSpecFile(openAPISpec)
		if err != nil {
			slog.Error("Error loading OpenAPI spec file", "error", err)
			return
		}

		err = usecases.AddService(serviceName, specData)
		if err != nil {
			slog.Error("Error adding service", "serviceName", serviceName, "error", err)
			return
		}
	},
}

func init() {
	AddCmd.Flags().StringVar(&openAPISpec, "spec", "", "OpenAPI spec filename (required)")
	AddCmd.MarkFlagRequired("spec")
}
