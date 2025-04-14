package generator

import (
	"dspn-regogenerator/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateServiceFolder(t *testing.T) {
	// Setup temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-service-folder")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after test

	serviceName := "testService"
	outputDir := tempDir

	// Call the function
	err = GenerateServiceFolder(serviceName, outputDir, "test", &policy.GeneralPolicies{
		Policies: []policy.PolicyClause{
			{
				UserPolicy: &policy.UserPolicy{
					PolicyDetail: policy.PolicyDetail{
						Value:    []string{"test"},
						Operator: "OR",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateServiceFolder returned an error: %v", err)
	}

	// Verify the service directory was created
	serviceDir := filepath.Join(outputDir, serviceName)
	if _, err := os.Stat(serviceDir); os.IsNotExist(err) {
		t.Errorf("Service directory %s was not created", serviceDir)
	}

	// Verify the OIDC file was created
	oidcFile := filepath.Join(serviceDir, "oidc.rego")
	if _, err := os.Stat(oidcFile); os.IsNotExist(err) {
		t.Errorf("OIDC file %s was not created", oidcFile)
	}

	// Verify the service file was created
	serviceFile := filepath.Join(serviceDir, "service.rego")
	if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
		t.Errorf("Service file %s was not created", serviceFile)
	}

	if t.Failed() {
		return
	}

	// Verify the contents of the OIDC file
	expectedContent := `package testService.oidc

import rego.v1

internal_keycloak_jwks_url := "http://keycloak:8080/keycloak/realms/teadal/protocol/openid-connect/certs"

jwks_preferred_urls := {
	"http://test": internal_keycloak_jwks_url,
	"https://test": internal_keycloak_jwks_url,
}

jwt_user_field_name := "email"

jwt_realm_access_field_name := "realm_access"

jwt_roles_field_name := "roles"
`
	content, err := os.ReadFile(oidcFile)
	if err != nil {
		t.Fatalf("Failed to read OIDC file: %v", err)
	}
	if string(content) != expectedContent {
		t.Errorf("OIDC file content does not match expected content.\nGot:\n%s\nExpected:\n%s", string(content), expectedContent)
	}

	// Verify the contents of the service file
	content, err = os.ReadFile(serviceFile)
	if err != nil {
		t.Fatalf("Failed to read service file: %v", err)
	}
	if !strings.Contains(string(content), "package testService") || !strings.Contains(string(content), "allow if") {
		t.Error("Service file content does not match expected content.")
	}
}
