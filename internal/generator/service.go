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
import data.input.attributes.request.http as request

# OIDC configuration discover url
metadata_url := "%s"

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

func generateOIDCfile(serviceName string, outputDir string, url string) error {
	data := fmt.Sprintf(oidcTemplate, serviceName, url)
	return os.WriteFile(outputDir+"/oidc.rego", []byte(data), 0644)
}

const serviceTemplate = `package %s

import rego.v1
import data.input.attributes.request.http as request
import data.%s.oidc.token

user := token.payload.preferred_username
roles := token.payload.realm_access.roles

# Generated access control policies
`

func generateServiceFile(serviceName string, outputDir string, policies *policy.GeneralPolicies) error {
	data := fmt.Sprintf(serviceTemplate, serviceName, serviceName)
	data += policies.ToRego()
	return os.WriteFile(outputDir+"/service.rego", []byte(data), 0644)
}
