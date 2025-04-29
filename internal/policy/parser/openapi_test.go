package parser_test

import (
	"dspn-regogenerator/internal/policy/parser"
	"os"
	"strings"
	"testing"
)

func TestParseOpenAPIPathsConditions(t *testing.T) {
	// Define a sample OpenAPI document
	cwd, _ := os.Getwd()
	cwd = strings.Split(cwd, "/internal")[0]
	os.Chdir(cwd)
	file, err := os.ReadFile("./testdata/schemas/httpbin-api.json")
	if err != nil {
		t.Fatalf("Failed to read OpenAPI file: %v", err)
	}
	// Parse the OpenAPI document
	r, err := parser.ParseOpenAPIPolicies(file)
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI file: %v", err)
	}
	if r == nil {
		t.Fatalf("Expected non-nil result, got nil")
	}
	if len(r.Policies) != 1 {
		t.Errorf("Expected 1 policy, got %d", len(r.Policies))
	}
	if r.Policies[0].RolePolicy != nil || r.Policies[0].UserPolicy != nil {
		t.Errorf("Expected nil RolePolicy and UserPolicy, got %v and %v", r.Policies[0].RolePolicy, r.Policies[0].UserPolicy)
	}
	if r.Policies[0].StorageLocationPolicy == nil ||
		r.Policies[0].StorageLocationPolicy.Operator != "OR" ||
		r.Policies[0].StorageLocationPolicy.Value[0] != "Europe" || r.Policies[0].StorageLocationPolicy.Value[1] != "USA" {
		t.Errorf("Expected non-nil StorageLocationPolicy, got nil")
	}
}

func TestParseOpenAPISpecializedPathConditions(t *testing.T) {
	// Define a sample OpenAPI document
	cwd, _ := os.Getwd()
	cwd = strings.Split(cwd, "/internal")[0]
	os.Chdir(cwd)
	file, err := os.ReadFile("./testdata/schemas/httpbin-api.json")
	if err != nil {
		t.Fatalf("Failed to read OpenAPI file: %v", err)
	}
	// Parse the OpenAPI document
	r, err := parser.ParseOpenAPIPolicies(file)
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI file: %v", err)
	}
	if r == nil {
		t.Fatalf("Expected non-nil result, got nil")
	}
	if len(r.SpecializedPaths) == 0 {
		t.Errorf("Expected specialized paths, got none")
	}
	if el, ok := r.SpecializedPaths["/anything"]; !ok || el.Path != "/anything" || el.Policies[0].RolePolicy != nil || el.Policies[0].UserPolicy != nil || el.Policies[0].StorageLocationPolicy != nil || el.Policies[0].CallPolicy.Value[0].Max != "10000" || el.Policies[0].CallPolicy.Value[0].UnitOfMeasure != "call_per_year" {
		t.Errorf("Expected specialized path /anything, got %v", el)
	}
}

func TestParseOpenAPISpecializedPathMethodConditions(t *testing.T) {
	cwd, _ := os.Getwd()
	cwd = strings.Split(cwd, "/internal")[0]
	os.Chdir(cwd)
	file, err := os.ReadFile("./testdata/schemas/httpbin-api.json")
	if err != nil {
		t.Fatalf("Failed to read OpenAPI file: %v", err)
	}
	// Parse the OpenAPI document
	r, err := parser.ParseOpenAPIPolicies(file)
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI file: %v", err)
	}
	if r == nil {
		t.Fatalf("Expected non-nil result, got nil")
	}
	if len(r.SpecializedPaths) == 0 {
		t.Errorf("Expected specialized paths, got none")
	}
	if el, ok := r.SpecializedPaths["/absolute-redirect/{n}"]; !ok {
		t.Errorf("Expected specialized path /anything, got %v", el)
	}
}

func TestParsing(t *testing.T) {
	cwd, _ := os.Getwd()
	cwd = strings.Split(cwd, "/internal")[0]
	os.Chdir(cwd)
	file, err := os.ReadFile("./testdata/schemas/httpbin-api.json")
	if err != nil {
		t.Fatalf("Failed to read OpenAPI file: %v", err)
	}
	url, err := parser.ParseOpenAPIIAM(file)
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI file: %v", err)
	}
	if url == nil {
		t.Fatalf("Expected non-nil result, got nil")
	}
	if *url != "http://localhost/keycloak/realms/master/.well-known/openid-configuration" {
		t.Errorf("Expected URL http://localhost/keycloak/realms/master/.well-known/openid-configuration, got %s", *url)
	}
}

func TestParseBearerPathRules(t *testing.T) {
	cwd, _ := os.Getwd()
	cwd = strings.Split(cwd, "/internal")[0]
	os.Chdir(cwd)
	file, err := os.ReadFile("./testdata/schemas/httpbin-api.json")
	if err != nil {
		t.Fatalf("Failed to read OpenAPI file: %v", err)
	}
	// Parse the OpenAPI document
	r, err := parser.ParseOpenAPIPolicies(file)
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI file: %v", err)
	}
	if _, ok := r.SpecializedPaths["/bearer"]; !ok {
		t.Errorf("Expected specialized path /bearer in parsed policies")
	}
	specPath := r.SpecializedPaths["/bearer"]
	if _, ok := specPath.SpecializedMethods["get"]; !ok {
		t.Errorf("Expected specialized method get in path /bearer")
	}
	if len(specPath.Policies) != 2 {
		t.Errorf("Expected 2 policies in path /bearer, got %d", len(specPath.Policies))
	}
	if specPath.Policies[0].RolePolicy == nil {
		t.Errorf("Expected non-nil RolePolicy in path /bearer")
	}
	if specPath.Policies[0].RolePolicy.Operator != "OR" {
		t.Errorf("Expected RolePolicy operator OR, got %s", specPath.Policies[0].RolePolicy.Operator)
	}
	if specPath.Policies[0].RolePolicy.Value[0] != "role3" {
		t.Errorf("Expected RolePolicy value role3, got %s", specPath.Policies[0].RolePolicy.Value[0])
	}
	if specPath.Policies[0].RolePolicy.Value[1] != "role2" {
		t.Errorf("Expected RolePolicy value role2, got %s", specPath.Policies[0].RolePolicy.Value[1])
	}

	if specPath.Policies[1].UserPolicy == nil {
		t.Errorf("Expected non-nil UserPolicy in path /bearer")
	}
	if specPath.Policies[1].UserPolicy.Operator != "OR" {
		t.Errorf("Expected UserPolicy operator OR, got %s", specPath.Policies[1].UserPolicy.Operator)
	}
	if specPath.Policies[1].UserPolicy.Value[0] != "user1@teadal.eu" {
		t.Errorf("Expected UserPolicy value user1@teadal.eu, got %s", specPath.Policies[1].UserPolicy.Value[0])
	}
	if specPath.Policies[1].UserPolicy.Value[1] != "user2@teadal.eu" {
		t.Errorf("Expected UserPolicy value user2@teadal.eu, got %s", specPath.Policies[1].UserPolicy.Value[1])
	}

	if specPath.Policies[1].StorageLocationPolicy == nil {
		t.Errorf("Expected non-nil StorageLocationPolicy in path /bearer")
	}
	if specPath.Policies[1].StorageLocationPolicy.Operator != "OR" {
		t.Errorf("Expected StorageLocationPolicy operator OR, got %s", specPath.Policies[1].StorageLocationPolicy.Operator)
	}
	if specPath.Policies[1].StorageLocationPolicy.Value[0] != "Italy" {
		t.Errorf("Expected StorageLocationPolicy value Europe, got %s", specPath.Policies[1].StorageLocationPolicy.Value[0])
	}
	if specPath.Policies[1].StorageLocationPolicy.Value[1] != "Spain" {
		t.Errorf("Expected StorageLocationPolicy value USA, got %s", specPath.Policies[1].StorageLocationPolicy.Value[1])
	}
}
