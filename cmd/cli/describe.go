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

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe [dsn]",
	Short: "Describe opa bundle referenced by dsn",
	Long:  "Describe opa bundle referenced by dsn.\n" + defaultDsnUsage,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo, path, err := getRepositoryAndPath(args)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		bundle, err := repo.Get(path)
		if err != nil {
			fmt.Printf("Error getting bundle %q: %v\n", path, err)
			return
		}
		fmt.Printf("Bundle %q:\n", path)
		fmt.Printf("  Services: %v\n", bundle.Describe())
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
}
