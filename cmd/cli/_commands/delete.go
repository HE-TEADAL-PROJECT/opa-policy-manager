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
	"log/slog"

	"github.com/spf13/cobra"
)

var ()

var DeleteCmd = &cobra.Command{
	Use:   "delete <service name>",
	Short: "Delete all policies related to a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]
		if serviceName == "" {
			cmd.Help()
			return
		}

		err := usecases.DeleteService(serviceName)
		if err != nil {
			slog.Error("Error deleting service", "serviceName", serviceName, "error", err)
		}
	},
}
