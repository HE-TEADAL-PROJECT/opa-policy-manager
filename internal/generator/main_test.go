package generator

import (
	"os"
	"testing"
)

func TestGenerateNewMain(t *testing.T) {
	serviceNames := []string{"service1", "service2"}
	outputDir := t.TempDir()

	err := GenerateNewMain(outputDir, serviceNames)
	if err != nil {
		t.Fatalf("failed to generate new main.rego: %v", err)
	}

	// Read the generated file and check its content
	content, err := os.ReadFile(outputDir + "/main.rego")
	if err != nil {
		t.Fatalf("failed to read generated main.rego: %v", err)
	}

	expectedContent := `package teadal
import data.service1.allow as service1_allow
import data.service2.allow as service2_allow
` + ImportMarker + `

default allow := false

allow if service1_allow
allow if service2_allow
` + RuleMarker + `
`
	if string(content) != expectedContent {
		t.Fatalf("generated main.rego content does not match expected content:\n=== GOT\n%s\n === WANT\n%s", string(content), expectedContent)
	}
}

func TestUpdateMainFile(t *testing.T) {
	serviceNames := []string{"service3", "service4"}
	outputDir := t.TempDir()

	// Create an initial main.rego file
	initialContent := `package teadal
import data.service1.allow as service1_allow
import data.service2.allow as service2_allow
` + ImportMarker + `

default allow := false

allow if service1_allow
allow if service2_allow
` + RuleMarker + `
`
	err := os.WriteFile(outputDir+"/main.rego", []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create initial main.rego: %v", err)
	}

	err = UpdateMainFile(outputDir, serviceNames)
	if err != nil {
		t.Fatalf("failed to update main.rego: %v", err)
	}

	content, err := os.ReadFile(outputDir + "/main.rego")
	if err != nil {
		t.Fatalf("failed to read updated main.rego: %v", err)
	}
	expectedContent := `package teadal
import data.service1.allow as service1_allow
import data.service2.allow as service2_allow
import data.service3.allow as service3_allow
import data.service4.allow as service4_allow
` + ImportMarker + `

default allow := false

allow if service1_allow
allow if service2_allow
allow if service3_allow
allow if service4_allow
` + RuleMarker + `
`
	if string(content) != expectedContent {
		t.Fatalf("updated main.rego content does not match expected content:\n=== GOT\n%s\n === WANT\n%s", string(content), expectedContent)
	}
}
