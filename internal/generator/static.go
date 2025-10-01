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

package generator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var StaticServiceNames = []string{"minio"}

// GenerateStaticFolders copy static folders from the PROJECT_ROOT/static to the outputDir path
func GenerateStaticFolders(outputDir string) {
	sourceDir := "./static"

	// Create the output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return
	}

	err := filepath.WalkDir(sourceDir, func(sourcePath string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path %s: %v\n", sourcePath, err)
			return err
		}

		// Construct the destination path
		relativePath, err := filepath.Rel(sourceDir, sourcePath)
		if err != nil {
			fmt.Printf("Error getting relative path for %s: %v\n", sourcePath, err)
			return err
		}
		destPath := filepath.Join(outputDir, relativePath)

		if d.IsDir() {
			// Create the destination directory if it doesn't exist
			if err := os.MkdirAll(destPath, 0755); err != nil {
				fmt.Printf("Error creating directory %s: %v\n", destPath, err)
				return err
			}
			fmt.Printf("Created directory: %s\n", destPath)
		} else {
			// It's a file, copy it
			sourceFile, err := os.Open(sourcePath)
			if err != nil {
				fmt.Printf("Error opening source file %s: %v\n", sourcePath, err)
				return err
			}
			defer sourceFile.Close()

			destFile, err := os.Create(destPath)
			if err != nil {
				fmt.Printf("Error creating destination file %s: %v\n", destPath, err)
				return err
			}
			defer destFile.Close()

			_, err = io.Copy(destFile, sourceFile)
			if err != nil {
				fmt.Printf("Error copying file from %s to %s: %v\n", sourcePath, destPath, err)
				return err
			}
			fmt.Printf("Copied file: %s -> %s\n", sourcePath, destPath)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %s: %v\n", sourceDir, err)
	}
}
