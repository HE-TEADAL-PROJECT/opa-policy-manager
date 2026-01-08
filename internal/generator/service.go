package generator

import (
	"bytes"
	"dspn-regogenerator/internal/policy"
	"fmt"
	"os"
	"text/template"
)

func GenerateServiceFolder(options ServiceOptions, outputDir string, IAMprovider string, policies *policy.GeneralPolicies) error {
	// Create the service directory
	serviceDir := outputDir + "/" + options.ServiceName
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return err
	}
	// create service specific files
	if err := generateOIDCfile(options.ServiceName, serviceDir, IAMprovider); err != nil {
		return fmt.Errorf("failed to generate OIDC file: %v", err)
	}
	if err := generateServiceFile(options, serviceDir, policies); err != nil {
		return fmt.Errorf("failed to generate service file: %v", err)
	}

	return nil
}

const oidcTemplate = `package %s.oidc

import rego.v1

request := input.attributes.request.http

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

#token := {"valid": valid, "payload": payload} if {
token := {"payload": payload} if {
	[_, encoded] := split(request.headers.authorization, " ")
    #[valid, _, payload] := io.jwt.decode_verify(encoded,{ "cert": json.marshal(jwks) })
	[_, payload, _] := io.jwt.decode(encoded)
}
`

func generateOIDCfile(serviceName string, outputDir string, url string) error {
	data := fmt.Sprintf(oidcTemplate, serviceName, url)
	return os.WriteFile(outputDir+"/oidc.rego", []byte(data), 0644)
}

const serviceTemplate = `package {{.ServiceName}}

import rego.v1
import data.{{.ServiceName}}.oidc.token

request := input.attributes.request.http

user := token.payload.preferred_username
roles contains role if some role in token.payload.realm_access.roles

{{ if .PathPrefix -}}
path := trim_prefix(request.path, "{{.PathPrefix}}")
{{- else -}}
path := request.path
{{- end }}
method := lower(request.method)

default allow := false
allow if {
	print("Params: ", {
		"request": request,
		"user": user,
		"roles": roles,
		"path": path,
		"method": method,
		"allow_request": allow_request
	})

	# Check if the user is authenticated
	#token.valid

	# Check if request is valid
	allow_request
}

default allow_request := false

# Generated access control policies
`

type ServiceOptions struct {
	ServiceName string
	PathPrefix  string
}

func generateServiceFile(serviceOptions ServiceOptions, outputDir string, policies *policy.GeneralPolicies) error {
	t := template.Must(template.New("service").Parse(serviceTemplate))
	buffer := &bytes.Buffer{}
	err := t.Execute(buffer, serviceOptions)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}
	data := buffer.String() + policies.ToRego()
	return os.WriteFile(outputDir+"/service.rego", []byte(data), 0644)
}
