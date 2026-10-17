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
	options := ServiceOptions{
		ServiceName: serviceName,
	}

	// Call the function
	err = GenerateServiceFolder(options, outputDir, "http://localhost:8000/keykloack/realms/test", &policy.GeneralPolicies{
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
	expectedOIDCFileContent := `package testService.oidc

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

#token := {"valid": valid, "payload": payload} if {
token := {"payload": payload} if {
	[_, encoded] := split(request.headers.authorization, " ")
    #[valid, _, payload] := io.jwt.decode_verify(encoded,{ "cert": json.marshal(jwks) })
	[_, payload, _] := io.jwt.decode(encoded)
}
`
	content, err := os.ReadFile(oidcFile)
	if err != nil {
		t.Fatalf("Failed to read OIDC file: %v", err)
	}
	if string(content) != expectedOIDCFileContent {
		t.Errorf("OIDC file content does not match expected content.\nGot:\n%s\nExpected:\n%s", string(content), expectedOIDCFileContent)
	}

	expectedServiceHeader := `package testService

import rego.v1
import data.testService.oidc.token

request := input.attributes.request.http

user := token.payload.preferred_username
roles contains role if some role in token.payload.realm_access.roles

path := split(request.path, "?")[0]
method := lower(request.method)

default allow := false`

	// Verify the contents of the service file
	content, err = os.ReadFile(serviceFile)
	if err != nil {
		t.Fatalf("Failed to read service file: %v", err)
	}
	if !strings.Contains(string(content), expectedServiceHeader) || !strings.Contains(string(content), "allow if") {
		t.Logf("Wanted header:\n%s\n", expectedServiceHeader)
		t.Logf("Got content:\n%s\n", string(content))
		t.Error("Service file content does not match expected content.")
	}
}

func TestGenerateServiceFolderWithPatPrefix(t *testing.T) {
	// Setup temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-service-folder")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after test

	serviceName := "testService"
	outputDir := tempDir
	options := ServiceOptions{
		ServiceName: serviceName,
		PathPrefix:  "/api/v1",
	}

	// Call the function
	err = GenerateServiceFolder(options, outputDir, "http://localhost:8000/keykloack/realms/test", &policy.GeneralPolicies{
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

	serviceDir := filepath.Join(outputDir, serviceName)
	serviceFile := filepath.Join(serviceDir, "service.rego")

	content, err := os.ReadFile(serviceFile)
	if err != nil {
		t.Fatalf("Failed to read service file: %v", err)
	}

	expectedContent := `path := split(trim_prefix(request.path, "/api/v1"), "?")[0]`
	if !strings.Contains(string(content), expectedContent) {
		t.Errorf("Service file content does not contain expected path prefix.\nGot:\n%s\nExpected:\n%s", string(content), expectedContent)
	}
}

func TestGenerateServiceFolderHyphenatedName(t *testing.T) {
	tempDir := t.TempDir()
	serviceName := "generated-petstore"
	options := ServiceOptions{
		ServiceName: serviceName,
	}

	err := GenerateServiceFolder(options, tempDir, "http://keycloak:8080/realms/test", &policy.GeneralPolicies{})
	if err != nil {
		t.Fatalf("GenerateServiceFolder failed: %v", err)
	}

	serviceDir := filepath.Join(tempDir, serviceName)
	serviceFile := filepath.Join(serviceDir, "service.rego")
	content, err := os.ReadFile(serviceFile)
	if err != nil {
		t.Fatalf("Failed to read service file: %v", err)
	}

	// Should contain package generated_petstore
	if !strings.Contains(string(content), "package generated_petstore") {
		t.Errorf("Expected package generated_petstore, got:\n%s", string(content))
	}
	// Should have path := split(request.path, "?")[0] without trim_prefix
	if !strings.Contains(string(content), `path := split(request.path, "?")[0]`) {
		t.Errorf("Expected path := split(request.path, \"?\")[0], got:\n%s", string(content))
	}
}
