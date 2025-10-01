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

package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const (
	binaryName = "opa-policy-manager"
	e2eDir     = "e2e"
	binaryPath = "./" + binaryName
)

// Get wd and return project roo
func getProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	dir = strings.Split(dir, e2eDir)[0]
	dir = strings.TrimRight(dir, "/")
	return dir
}

func TestMain(m *testing.M) {
	// Build the CLI application
	targetFile := getProjectRoot() + "/cmd/cli"
	cmd := exec.Command("go", "build", "-o", binaryName, targetFile)
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// If build fails, print the error and exit
		fmt.Fprint(os.Stderr, stderr.String())
		os.Exit(1)
	}

	os.Chmod(binaryName, 0755) // Ensure the binary is executable

	// Run the tests
	exitCode := m.Run()
	// Clean up the built binary
	_ = exec.Command("rm", binaryName).Run()
	// Exit with the appropriate code
	os.Exit(exitCode)
}
