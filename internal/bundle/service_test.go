package bundle

import (
	"dspn-regogenerator/internal/policy"
	policygenerator "dspn-regogenerator/internal/policy/generator"
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestGenerateServiceFile(t *testing.T) {
	service := Service{
		name:    "test_service",
		oidcUrl: "https://example.com/oidc",
		policy: policy.GeneralPolicies{
			Policies: []policy.PolicyClause{
				{
					UserPolicy: &policy.UserPolicy{
						Operator: policy.OperatorAnd,
						EnumeratedValue: policy.EnumeratedValue{
							Value: []string{"user1", "user2"},
						},
					},
				},
			},
		},
	}

	got, err := service.generateServiceFiles()
	if err != nil {
		t.Fatalf("GenerateServiceFiles() error = %v", err)
	}
	if serviceFile, ok := got["/test_service/service.rego"]; !ok {
		t.Errorf("GenerateServiceFiles() did not return service.rego file")
	} else {
		if !strings.Contains(serviceFile, "package test_service") {
			t.Errorf("GenerateServiceFiles() did not contain expected package declaration")
		}
		if !strings.Contains(serviceFile, policygenerator.RequestPolicyName+" :=") {
			t.Errorf("GenerateServiceFiles() did not contain expected global rule name %s", policygenerator.RequestPolicyName)
		}
		if t.Failed() {
			t.Log("Generated service.rego content:\n" + serviceFile)
		}
	}

	if oidcFile, ok := got["/test_service/oidc.rego"]; !ok {
		t.Errorf("GenerateServiceFiles() did not return oidc.rego file")
	} else {
		if !strings.Contains(oidcFile, "package test_service.oidc") {
			t.Errorf("GenerateServiceFiles() did not contain expected package declaration for OIDC")
		}
		if !strings.Contains(oidcFile, "metadata_url := \"https://example.com/oidc\"") {
			t.Errorf("GenerateServiceFiles() did not contain expected metadata URL")
		}
		if t.Failed() {
			t.Log("Generated oidc.rego content:\n" + oidcFile)
		}
	}
	if len(got) != 2 {
		t.Errorf("GenerateServiceFiles() returned %d files, expected 2\n%v", len(got), slices.Collect(maps.Keys(got)))
	}
}
