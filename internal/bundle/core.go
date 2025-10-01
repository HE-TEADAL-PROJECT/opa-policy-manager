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
	"context"
	policy "dspn-regogenerator/internal/policy"
	"dspn-regogenerator/internal/policy/parser"
	"fmt"
	"slices"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	opabundle "github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/compile"
)

const mainFilePath = "/main.rego"
const mainPackage = "envoy.authz"
const metadataServicesKey = "services"

// Represent a single generated service in the bundle
type Service struct {
	name    string
	oidcUrl string
	policy  policy.GeneralPolicies
}

// Create a new service with the given name and specification
func NewService(name string, spec *parser.ServiceSpec) *Service {
	return &Service{
		name:    name,
		policy:  spec.Policies,
		oidcUrl: spec.IdentityProvider,
	}
}

// A collection of rego files and metadata that represent all services files
type Bundle struct {
	bundle       *opabundle.Bundle
	serviceNames []string
}

// Add or update the files associated to a service in the bundle
func (b *Bundle) AddService(service Service) error {
	isNewService := !slices.Contains(b.serviceNames, service.name)

	// Generate new service files
	newFiles, err := service.generateServiceFiles()
	if err != nil {
		return fmt.Errorf("failed to generate service files for %s: %w", service.name, err)
	}

	// Generate main files
	if isNewService {
		newFiles[mainFilePath] = generateMainFile(append(b.serviceNames, service.name))
	}

	// Parse files
	newModules, err := compileServiceFiles(newFiles)
	if err != nil {
		return fmt.Errorf("failed to compile service files for %s: %w", service.name, err)
	}

	// Merge the new bundle into the existing one
	if isNewService {
		// Update manifest metadata
		b.bundle.Manifest.Metadata[metadataServicesKey] = append(b.serviceNames, service.name)
		newRoots := append(*b.bundle.Manifest.Roots, service.name)
		b.bundle.Manifest.Roots = &newRoots
	}
	if slices.Contains(b.serviceNames, service.name) {
		// Fetch existing files and update them
		for path, content := range newFiles {
			for i, existingFile := range b.bundle.Modules {
				if existingFile.Path == path {
					b.bundle.Modules[i].Raw = []byte(content)
					b.bundle.Modules[i].Parsed = newModules[path]
					break
				}
			}
		}
	} else if len(newFiles) > 0 {
		// Add new files to the bundle
		for path, content := range newFiles {
			if path == mainFilePath {
				// Skip main.rego here, it will be updated later
				continue
			}
			b.bundle.Modules = append(b.bundle.Modules, opabundle.ModuleFile{
				URL:    path,
				Path:   path,
				Raw:    []byte(content),
				Parsed: newModules[path],
			})
		}
		b.serviceNames = append(b.serviceNames, service.name)
	}

	// update the main.rego file
	for i, module := range b.bundle.Modules {
		if module.Path == mainFilePath {
			b.bundle.Modules[i].Raw = []byte(newFiles[mainFilePath])
			b.bundle.Modules[i].Parsed = newModules[mainFilePath]
			break
		}
	}

	compiler := compile.New()
	compiler.WithBundle(b.bundle)
	err = compiler.Build(context.Background())
	if err != nil {
		return fmt.Errorf("failed to build bundle after adding service %s: %w", service.name, err)
	}
	b.bundle = compiler.Bundle()

	return nil
}

// Remove the file associated to a service from the bundle
func (b *Bundle) RemoveService(serviceName string) error {
	for i, name := range b.serviceNames {
		if name == serviceName {
			b.serviceNames = append(b.serviceNames[:i], b.serviceNames[i+1:]...)
			break
		}
	}

	newModules := make([]opabundle.ModuleFile, 0, len(b.bundle.Modules))
	for _, module := range b.bundle.Modules {
		if strings.HasPrefix(module.Path, "/"+serviceName) || module.Path == mainFilePath {
			// Skip modules that belong to the service being removed
			continue
		}
		newModules = append(newModules, module)
	}
	mainFileContent := generateMainFile(b.serviceNames)
	newFiles, err := compileServiceFiles(map[string]string{
		mainFilePath: mainFileContent,
	})
	if err != nil {
		return fmt.Errorf("failed to compile main.rego after removing service %s: %w", serviceName, err)
	}
	newModules = append(newModules, opabundle.ModuleFile{
		URL:    mainFilePath,
		Path:   mainFilePath,
		Raw:    []byte(mainFileContent),
		Parsed: newFiles[mainFilePath],
	})

	b.bundle.Modules = newModules
	b.bundle.Manifest.Metadata[metadataServicesKey] = b.serviceNames
	newRoots := make([]string, 0, len(*b.bundle.Manifest.Roots))
	for _, root := range *b.bundle.Manifest.Roots {
		if root != serviceName {
			newRoots = append(newRoots, root)
		}
	}
	b.bundle.Manifest.Roots = &newRoots

	compiler := compile.New()
	compiler.WithBundle(b.bundle)
	err = compiler.Build(context.Background())
	if err != nil {
		return fmt.Errorf("failed to build bundle after removing service %s: %w", serviceName, err)
	}
	b.bundle = compiler.Bundle()

	return nil
}

// Return the list of services in the bundle
func (b *Bundle) Describe() []string {
	return b.serviceNames
}

// Load an empty bundle with the service refo files
func New(service *Service) (*Bundle, error) {
	files, err := service.generateServiceFiles()
	if err != nil {
		return nil, err
	}
	files[mainFilePath] = generateMainFile([]string{service.name})

	manifest := opabundle.Manifest{}
	manifest.Init()
	manifest.Metadata = make(map[string]any)
	manifest.Metadata[metadataServicesKey] = []string{service.name}
	manifest.Roots = &[]string{service.name, "envoy"}

	modules, err := compileServiceFiles(files)
	if err != nil {
		return nil, fmt.Errorf("failed to compile service files: %w", err)
	}

	bundleFiles := make([]opabundle.ModuleFile, 0, len(files))
	for path, content := range files {
		bundleFiles = append(bundleFiles, opabundle.ModuleFile{
			URL:    path,
			Path:   path,
			Raw:    []byte(content),
			Parsed: modules[path],
		})
	}

	bundle := opabundle.Bundle{
		Manifest: manifest,
		Modules:  bundleFiles,
		Data:     map[string]any{},
	}

	compiler := compile.New()
	compiler.WithBundle(&bundle)
	err = compiler.Build(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to build bundle: %w", err)
	}

	return &Bundle{
		bundle:       compiler.Bundle(),
		serviceNames: []string{service.name},
	}, nil
}

func compileServiceFiles(files map[string]string) (map[string]*ast.Module, error) {
	modules := make(map[string]*ast.Module)
	for path, content := range files {
		module := ast.MustParseModule(content)
		modules[path] = module
	}
	compiler := ast.NewCompiler()
	compiler.Compile(modules)
	if compiler.Failed() {
		return nil, fmt.Errorf("failed to compile service files: %v", compiler.Errors)
	}
	return modules, nil
}
