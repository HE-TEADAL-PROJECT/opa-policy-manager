package generator

import (
	"dspn-regogenerator/internal/policy"
	"fmt"
	"os"
)

func GenerateServiceFolder(serviceName string, outputDir string, IAMprovider string, policies *policy.GeneralPolicies) error {
	// Create the service directory
	serviceDir := outputDir + "/" + serviceName
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return err
	}
	// create service specific files
	if err := generateOIDCfile(serviceName, serviceDir, IAMprovider); err != nil {
		return fmt.Errorf("failed to generate OIDC file: %v", err)
	}
	if err := generateServiceFile(serviceName, serviceDir, policies); err != nil {
		return fmt.Errorf("failed to generate service file: %v", err)
	}

	return nil
}

const oidcTemplate = `package %s.oidc

import rego.v1

internal_keycloak_jwks_url := "http://keycloak:8080/keycloak/realms/teadal/protocol/openid-connect/certs"

jwks_preferred_urls := {
	"http://%s": internal_keycloak_jwks_url,
	"https://%s": internal_keycloak_jwks_url,
}

jwt_user_field_name := "email"

jwt_realm_access_field_name := "realm_access"

jwt_roles_field_name := "roles"
`

func generateOIDCfile(serviceName string, outputDir string, url string) error {
	data := fmt.Sprintf(oidcTemplate, serviceName, url, url)
	return os.WriteFile(outputDir+"/oidc.rego", []byte(data), 0644)
}

const serviceTemplate = `package %s

import rego.v1

`

func generateServiceFile(serviceName string, outputDir string, policies *policy.GeneralPolicies) error {
	data := fmt.Sprintf(serviceTemplate, serviceName)
	data += policies.ToRego()
	return os.WriteFile(outputDir+"/service.rego", []byte(data), 0644)
}
