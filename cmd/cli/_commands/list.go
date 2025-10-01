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

	"github.com/spf13/cobra"
)

var ReservedServiceName = []string{}

var (
	verbose bool
)

var ListCmd = &cobra.Command{
	Use:   "list [-v]",
	Short: "List all available services",
	Long:  `List all available services in the bundle. Use the verbose flag for detailed list of rego files.`,
	Run: func(cmd *cobra.Command, args []string) {
		bundleStructure, err := usecases.GetBundleStructure()
		if err != nil {
			slog.Error("Error getting bundle structure", "error", err)
			return
		}
		slog.Info(fmt.Sprintf("Services: %v", bundleStructure.Services))
		if verbose {
			slog.Info(fmt.Sprintf("Files: %v", bundleStructure.Files))
		}
	},
}

func init() {
	ListCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}
