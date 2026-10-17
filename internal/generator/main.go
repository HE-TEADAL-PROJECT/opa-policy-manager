package generator

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

const ImportMarker = "# generator:import"
const RuleMarker = "# generator:rule"

var mainTemplate = `package teadal

import rego.v1

%s
default allow := false

request_host := host if {
	raw := input.attributes.request.http.host
	no_port := split(raw, ":")[0]
	host := split(no_port, ".")[0]
} else := host if {
	raw := input.attributes.request.http.headers[":authority"]
	no_port := split(raw, ":")[0]
	host := split(no_port, ".")[0]
} else := ""

%s`

// ToPackageName converts a service name to a valid Rego package identifier by replacing hyphens and invalid chars with underscores.
func ToPackageName(serviceName string) string {
	var b strings.Builder
	for i, r := range serviceName {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	res := b.String()
	if len(res) == 0 {
		return "service"
	}
	if res[0] >= '0' && res[0] <= '9' {
		return "_" + res
	}
	return res
}

func generateImports(serviceNames []string) string {
	imports := ""
	for _, serviceName := range serviceNames {
		pkg := ToPackageName(serviceName)
		imports += fmt.Sprintf(`import data.%s.allow as %s_allow`, pkg, pkg) + "\n"
	}
	imports += ImportMarker + "\n"
	return imports
}

func generateRules(serviceNames []string) string {
	rules := ""
	for _, serviceName := range serviceNames {
		pkg := ToPackageName(serviceName)
		rules += fmt.Sprintf("allow if {\n\trequest_host == %q\n\t%s_allow\n}\n", serviceName, pkg)
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
