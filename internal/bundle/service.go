package bundle

import (
	policy "dspn-regogenerator/internal/policy"
	policygenerator "dspn-regogenerator/internal/policy/generator"
	"dspn-regogenerator/internal/policy/parser"
	"fmt"
	"strings"
	"text/template"
)

type Service struct {
	name    string
	oidcUrl string
	policy  policy.GeneralPolicies
}

type ServiceTemplateData struct {
	ServiceName    string
	PathPrefix     string
	GlobalRuleName string
}

var serviceTemplate = template.Must(template.New("service").Parse(`package {{.ServiceName}}

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
	# Check if the user is authenticated
	token.valid

	# Check if request is valid
	{{.GlobalRuleName}}
}
# Generated access control policies
`))

type OIDDTemplateData struct {
	ServiceName string
	MetadataURL string
}

var oidcTemplate = template.Must(template.New("oidc").Parse(`package {{.ServiceName}}.oidc

import rego.v1

request := input.attributes.request.http

# OIDC configuration discover url
metadata_url := "{{.MetadataURL}}"

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
`))

func (s *Service) generateServiceFiles() (map[string]string, error) {
	files := make(map[string]string)
	// Generate service files based on the policy
	policies, err := policygenerator.GenerateServiceRego(policygenerator.ServiceData{}, s.policy)
	if err != nil {
		return nil, fmt.Errorf("impossible to generate policies code for service %s: %w", s.name, err)
	}

	// Generate the service file
	data := ServiceTemplateData{
		ServiceName:    s.name,
		PathPrefix:     "/" + s.name,
		GlobalRuleName: policygenerator.RequestPolicyName,
	}
	builder := strings.Builder{}
	if err := serviceTemplate.Execute(&builder, data); err != nil {
		return nil, fmt.Errorf("failed to execute service template: %w", err)
	}
	builder.WriteString(policies)
	files["/"+s.name+"/service.rego"] = builder.String()

	// Generate the OIDC file
	oidcData := OIDDTemplateData{
		ServiceName: s.name,
		MetadataURL: s.oidcUrl,
	}
	oidcBuilder := strings.Builder{}
	if err := oidcTemplate.Execute(&oidcBuilder, oidcData); err != nil {
		return nil, fmt.Errorf("failed to execute OIDC template: %w", err)
	}
	files["/"+s.name+"/oidc.rego"] = oidcBuilder.String()

	return files, nil
}

func NewService(name string, spec *parser.ServiceSpec) *Service {
	return &Service{
		name:    name,
		policy:  spec.Policies,
		oidcUrl: spec.IdentityProvider,
	}
}
