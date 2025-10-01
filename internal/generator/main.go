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

// Package generator provides functionality to generate or update the rego files for the each openapi service.
//
// Each service is composed by a collection of rego files organized in a single flat folder.
//   - the `service.rego` file contains the rules for the service
//   - the `identity.rego` file contains the authorization rules for the service
package generator

import (
	"bytes"
	"fmt"
	"os"
)

const ImportMarker = "# generator:import"
const RuleMarker = "# generator:rule"

var mainTemplate = `package teadal
%s
default allow := false

%s`

func generateImports(serviceNames []string) string {
	imports := ""
	for _, serviceName := range serviceNames {
		imports += fmt.Sprintf(`import data.%s.allow as %s_allow`, serviceName, serviceName) + "\n"
	}
	imports += ImportMarker + "\n"
	return imports
}

func generateRules(serviceNames []string) string {
	rules := ""
	for _, serviceName := range serviceNames {
		rules += fmt.Sprintf(`allow if %s_allow`, serviceName) + "\n"
	}
	rules += RuleMarker + "\n"
	return rules
}

func GenerateNewMain(outputDir string, serviceNames []string) error {
	// Create the main.go file with the new content
	mainContent := fmt.Sprintf(mainTemplate, generateImports(serviceNames), generateRules(serviceNames))
	return os.WriteFile(outputDir+"/main.rego", []byte(mainContent), 0644)
}

func UpdateMainFile(outputDir string, serviceNames []string) error {
	// Read the existing main.go file
	existingContent, err := os.ReadFile(outputDir + "/main.rego")
	if err != nil {
		return fmt.Errorf("failed to read main.rego: %v", err)
	}

	// Subsitute the import and rule markers with the new content
	newImports := generateImports(serviceNames)
	newRules := generateRules(serviceNames)

	newContent := bytes.Replace(existingContent, []byte(ImportMarker+"\n"), []byte(newImports), 1)
	newContent = bytes.Replace(newContent, []byte(RuleMarker+"\n"), []byte(newRules), 1)

	// Write the new content to the main.go file
	return os.WriteFile(outputDir+"/main.rego", []byte(newContent), 0644)
}
