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
	"context"
	"dspn-regogenerator/internal/config"
	"dspn-regogenerator/internal/usecases"

	"github.com/spf13/cobra"
)

var TestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the bundle",
	Long:  `Test the bundle against the httpbin service. This will run all the tests in the bundle and print the results.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		// Print the current configuration
		cmd.Println("Configuration:")
		cmd.Println("    MinIO Endpoint:", config.MinioEndpoint)
		cmd.Println("    MinIO Access Key:", config.MinioAccessKey)
		cmd.Println("    MinIO Secret Key:", config.MinioSecretKey)
		cmd.Println("    MinIO Bucket:", config.MinioBucket)
		cmd.Println("    MinIO Bundle Prefix:", config.MinioBundlePrefix)
		cmd.Println("    MinIO Timeout:", config.MinioTimeout)

		if err := usecases.InitialTest(ctx); err != nil {
			cmd.PrintErrf("Error during initial test: %v\n", err)
		}
	},
}
