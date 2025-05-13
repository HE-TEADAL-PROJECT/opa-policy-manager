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
	err = GenerateServiceFolder(serviceName, outputDir, "http://localhost:8000/keykloack/realms/test", &policy.GeneralPolicies{
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

request := input.attributes.request.http

# OIDC configuration discover url
metadata_url := "http://localhost:8000/keykloack/realms/test"

# Generate code 

metadata := http.send({
    "url": metadata_url,
    "method": "GET",
    "headers": {
        "accept": "application/json"
    },
    "force_cache": true,
    "force_cache_duration_seconds": 86400 # Cache response for 24 hours
}).body

jwks_uri := metadata.jwks_uri

jwks := http.send({
    "url": jwks_uri,
    "method": "GET",
    "headers": {
        "accept": "application/json"
    },
    "force_cache": true,
    "force_cache_duration_seconds": 3600 # Cache response for 1 hour
}).body

encoded := split(request.headers.authorization, " ")[1]

token := {"valid": valid, "payload": payload} if {
    [_, encoded] := split(request.headers.authorization, " ")
    [valid, _, payload] := io.jwt.decode_verify(encoded,{ "cert": json.marshal(jwks) })
}
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
