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
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateStaticFolders(t *testing.T) {
	// Setup temporary directories for testing
	sourceDir := "./static"
	outputDir := t.TempDir()

	// Create a mock static directory structure
	err := os.MkdirAll(filepath.Join(sourceDir, "subdir"), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock source directory: %v", err)
	}
	defer os.RemoveAll(sourceDir) // Cleanup after test

	// Create mock files in the static directory
	mockFilePath := filepath.Join(sourceDir, "file1.txt")
	err = os.WriteFile(mockFilePath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create mock file: %v", err)
	}

	mockSubFilePath := filepath.Join(sourceDir, "subdir", "file2.txt")
	err = os.WriteFile(mockSubFilePath, []byte("subdir content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create mock subdirectory file: %v", err)
	}

	// Call the function to test
	GenerateStaticFolders(outputDir)

	// Verify the output directory structure
	expectedFilePath := filepath.Join(outputDir, "file1.txt")
	if _, err := os.Stat(expectedFilePath); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedFilePath)
	}

	expectedSubFilePath := filepath.Join(outputDir, "subdir", "file2.txt")
	if _, err := os.Stat(expectedSubFilePath); os.IsNotExist(err) {
		t.Errorf("Expected file %s does not exist", expectedSubFilePath)
	}

	// Verify file contents
	content, err := os.ReadFile(expectedFilePath)
	if err != nil {
		t.Errorf("Failed to read file %s: %v", expectedFilePath, err)
	}
	if string(content) != "test content" {
		t.Errorf("Unexpected content in file %s: got %s, want %s", expectedFilePath, string(content), "test content")
	}

	subContent, err := os.ReadFile(expectedSubFilePath)
	if err != nil {
		t.Errorf("Failed to read file %s: %v", expectedSubFilePath, err)
	}
	if string(subContent) != "subdir content" {
		t.Errorf("Unexpected content in file %s: got %s, want %s", expectedSubFilePath, string(subContent), "subdir content")
	}
}
