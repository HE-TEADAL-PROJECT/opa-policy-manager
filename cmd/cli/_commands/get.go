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
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"log/slog"

	"github.com/spf13/cobra"
)

var (
	outputDir string
)

func init() {
	GetCmd.Flags().StringVar(&outputDir, "output", "./output", "Output path for the latest bundle")
}

var GetCmd = &cobra.Command{
	Use:   "get [--output <path>]",
	Short: "Get latest bundle",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the latest bundle
		minioRepo, err := bundle.NewMinioRepositoryFromConfig()
		if err != nil {
			slog.Error("Error creating minio repository", "error", err)
			return
		}
		b, err := minioRepo.Read(config.LatestBundleName)
		if err != nil {
			slog.Error("Error reading bundle from Minio", "error", err)
			return
		}
		fileRepo := bundle.NewFileSystemRepository(outputDir)
		if err := fileRepo.Write(config.LatestBundleName, *b); err != nil {
			slog.Error("Error writing bundle to file system", "error", err)
			return
		}
		slog.Info("Bundle written to file system successfully", "path", outputDir)
	},
}
