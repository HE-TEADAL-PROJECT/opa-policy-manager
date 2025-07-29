package bundle

import (
	policygen "dspn-regogenerator/internal/policy/generator"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	opabundle "github.com/open-policy-agent/opa/v1/bundle"
)

const mainFilePath = "/main.rego"

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
	return fmt.Sprintf("package main\n\n%s\ndefault allow := false\n\n%s\n", strings.Join(imports, "\n"), strings.Join(allowRules, "\n"))
}

// A collection of rego files and metadata that represent all services files
type Bundle struct {
	bundle       *opabundle.Bundle
	serviceNames []string
}

// Add or update the files associated to a service in the bundle
func (b *Bundle) AddService(service Service) error {
	newFiles, err := service.generateServiceFiles()
	if err != nil {
		return fmt.Errorf("failed to generate service files for %s: %w", service.name, err)
	}
	newFiles[mainFilePath] = generateMainFile(append(b.serviceNames, service.name))
	newModules, err := compileServiceFiles(newFiles)
	if err != nil {
		return fmt.Errorf("failed to compile service files for %s: %w", service.name, err)
	}

	// Merge the new bundle into the existing one
	b.bundle.Manifest.Metadata["services"] = append(b.serviceNames, service.name)
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
	b.bundle.Manifest.Metadata["services"] = b.serviceNames

	return nil
}

// Load an empty bundle with the service refo files
func NewFromService(service Service) (*Bundle, error) {
	files, err := service.generateServiceFiles()
	if err != nil {
		return nil, err
	}
	files[mainFilePath] = generateMainFile([]string{service.name})

	manifest := opabundle.Manifest{}
	manifest.Init()
	manifest.Metadata = make(map[string]interface{})
	manifest.Metadata["services"] = []string{service.name}
	manifest.Roots = &[]string{service.name}

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
	}

	return &Bundle{
		bundle:       &bundle,
		serviceNames: []string{service.name},
	}, nil
}

// Load a bundle from a reader of a tar.gz file
func NewFromTarball(reader io.Reader) (*Bundle, error) {
	loader := opabundle.NewTarballLoader(reader)
	bundleReader := opabundle.NewCustomReader(loader)
	bundle, err := bundleReader.Read()
	if err != nil {
		return nil, fmt.Errorf("impossible to read bundle form tarball archive: %w", err)
	}

	// Load from the bundle manifest metadata the service names
	if bundle.Manifest.Metadata == nil || bundle.Manifest.Metadata["services"] == nil {
		return nil, fmt.Errorf("bundle manifest metadata does not contain 'services' key")
	}
	array := bundle.Manifest.Metadata["services"].([]interface{})
	serviceNames := make([]string, 0, len(array))
	for _, v := range array {
		if serviceName, ok := v.(string); ok {
			serviceNames = append(serviceNames, serviceName)
		} else {
			return nil, fmt.Errorf("invalid service name in bundle manifest metadata: %v", v)
		}
	}

	return &Bundle{
		bundle:       &bundle,
		serviceNames: serviceNames,
	}, nil
}

// Repository is an interface for writing bundle to a storage system.
type Repository interface {
	// Write a bundle to the repository, returning an error if it fails.
	Save(path string, bundle Bundle) error

	// Read reads the bundle from the repository, returning the bundle and an error if it fails.
	Get(path string) (*Bundle, error)
}
