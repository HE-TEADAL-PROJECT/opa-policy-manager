// Copyright 2025 Matteo Brambilla - TEADAL
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bundle

import (
	policygen "dspn-regogenerator/internal/policy/generator"
	policygenerator "dspn-regogenerator/internal/policy/generator"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
)

type ServiceTemplateData struct {
	ServiceName    string
	PathPrefix     string
	GlobalRuleName string
}

type OIDDTemplateData struct {
	ServiceName string
	MetadataURL string
}

//go:embed templates/service.rego.tmpl
var serviceTemplateContent string

//go:embed templates/oidc.rego.tmpl
var oidcTemplateContent string

var serviceTemplate = template.Must(template.New("service").Parse(serviceTemplateContent))
var oidcTemplate = template.Must(template.New("oidc").Parse(oidcTemplateContent))

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

func generateMainFile(serviceNames []string) string {
	// Generate a main.rego file that imports all service files
	imports := make([]string, len(serviceNames))
	for i, name := range serviceNames {
		imports[i] = fmt.Sprintf("import data.%s", name)
	}
	allowRules := make([]string, len(serviceNames))
	for i, name := range serviceNames {
		allowRules[i] = fmt.Sprintf("allow if %s.%s", name, policygen.RequestPolicyName)
	}
	return fmt.Sprintf("package %s\n\n%s\ndefault allow := false\n\n%s\n", mainPackage, strings.Join(imports, "\n"), strings.Join(allowRules, "\n"))
}
