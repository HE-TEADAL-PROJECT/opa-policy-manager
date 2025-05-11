package bundle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	opabundle "github.com/open-policy-agent/opa/v1/bundle"
)

// Represent a OPA bundle in the teadal context, which is a collection of services identified by an unique name. Each service may contain multiple rego file and it is stored in a directory wit its name.
type Bundle struct {
	bundle *opabundle.Bundle
}

const mainFilePath = "/main.rego"

// NewFromFS creates a new Bundle from a file system. The file system should contain the OPA bundle files.
func NewFromFS(ctx context.Context, fs fs.FS, serviceNames ...string) (*Bundle, error) {
	// Load the fs
	opab, err := opabundle.NewCustomReader(opabundle.NewFSLoaderWithRoot(fs, ".")).Read()
	if err != nil {
		return nil, err
	}

	opab.Manifest.Metadata = map[string]interface{}{"services": serviceNames}

	return &Bundle{&opab}, nil
}

// Read the bundle metadata service key, which is a list of service names. Modify it to be a list of strings if it is not already.
// This is done to ensure that the metadata is in a consistent format.
func (b *Bundle) normalizeMetadata() error {
	if b.bundle.Manifest.Metadata == nil {
		return errors.New("no services metadata found in the bundle")
	}
	if serviceList, ok := b.bundle.Manifest.Metadata["services"]; !ok {
		return errors.New("no services metadata found in the bundle")
	} else {
		if _, ok := serviceList.([]string); ok {
			return nil
		} else if services, ok := serviceList.([]interface{}); ok {
			servicesStr := make([]string, len(services))
			for i, service := range services {
				if serviceStr, ok := service.(string); ok {
					servicesStr[i] = serviceStr
				} else {
					return errors.New("invalid service name in metadata")
				}
			}
			b.bundle.Manifest.Metadata["services"] = servicesStr
			return nil
		}
		return errors.New("invalid services metadata format")
	}
}

// NewFromArchive creates a new Bundle from an archive reader. The reader should be a tarball containing the OPA bundle files.
func NewFromArchive(ctx context.Context, reader io.Reader) (*Bundle, error) {
	loader := opabundle.NewTarballLoaderWithBaseURL(reader, "")
	bundle, err := opabundle.NewCustomReader(loader).Read()
	if err != nil {
		return nil, err
	}

	newBundle := Bundle{&bundle}
	if err := newBundle.normalizeMetadata(); err != nil {
		return nil, err
	}

	return &newBundle, nil
}

// Return the list of services present in the bundle. The services are identified by their names.
func (b *Bundle) Services() ([]string, error) {
	if serviceList, ok := b.bundle.Manifest.Metadata["services"]; !ok {
		return nil, errors.New("no services metadata found in the bundle")
	} else {
		if services, ok := serviceList.([]string); ok {
			return services, nil
		} else if services, ok := serviceList.([]interface{}); ok {
			servicesStr := make([]string, len(services))
			for i, service := range services {
				if serviceStr, ok := service.(string); ok {
					servicesStr[i] = serviceStr
				} else {
					return nil, errors.New("invalid service name in metadata")
				}
			}
			return servicesStr, nil
		}
		return nil, errors.New("invalid services metadata format")
	}
}

func (b *Bundle) AddService(serviceName string, specData map[string][]byte) error {
	// Add the service to the bundle metadata
	if b.bundle.Manifest.Metadata == nil {
		b.bundle.Manifest.Metadata = make(map[string]interface{})
	}
	if services, ok := b.bundle.Manifest.Metadata["services"]; ok {
		if serviceList, ok := services.([]string); ok {
			if !slices.Contains(b.bundle.Manifest.Metadata["services"].([]string), serviceName) {
				b.bundle.Manifest.Metadata["services"] = append(serviceList, serviceName)
			}
		} else if serviceList, ok := services.([]interface{}); ok {
			if !slices.Contains(serviceList, interface{}(serviceName)) {
				b.bundle.Manifest.Metadata["services"] = append(serviceList, serviceName)
			}
		} else {
			return errors.New("invalid services metadata format")
		}
	} else {
		b.bundle.Manifest.Metadata["services"] = []string{serviceName}
	}

	// Add the spec data files to the bundle
	for path, data := range specData {
		cleanPath := filepath.Clean(path)
		if cleanPath[0] != os.PathSeparator && cleanPath[0] != '.' {
			cleanPath = string(os.PathSeparator) + cleanPath
		}

		parsedData, err := ast.ParseModule(cleanPath, string(data))
		if err != nil {
			return fmt.Errorf("failed to parse module %s: %w", cleanPath, err)
		}

		moduleFound := false
		for index, module := range b.bundle.Modules {
			if module.Path == cleanPath {
				module.Raw = data
				module.Parsed = parsedData
				b.bundle.Modules[index] = module
				moduleFound = true
			}
		}
		if !moduleFound {
			b.bundle.Modules = append(b.bundle.Modules, opabundle.ModuleFile{
				URL:    cleanPath,
				Path:   cleanPath,
				Raw:    data,
				Parsed: parsedData,
			})
		}
	}

	return nil
}

func (b *Bundle) GetMain() ([]byte, error) {
	if b.bundle == nil {
		return nil, errors.New("bundle is nil")
	}
	if len(b.bundle.Modules) == 0 {
		return nil, errors.New("no modules found in the bundle")
	}

	for _, module := range b.bundle.Modules {
		if module.Path == mainFilePath {
			return module.Raw, nil
		}
	}

	return nil, errors.New("main.rego not found in the bundle")
}

func (b *Bundle) RemoveService(serviceName string) error {
	// Remove the service from the bundle metadata
	if b.bundle.Manifest.Metadata == nil {
		return errors.New("no services metadata found in the bundle")
	}
	if services, ok := b.bundle.Manifest.Metadata["services"]; ok {
		if serviceList, ok := services.([]string); ok {
			for i, service := range serviceList {
				if service == serviceName {
					b.bundle.Manifest.Metadata["services"] = append(serviceList[:i], serviceList[i+1:]...)
					break
				}
			}
		} else {
			return errors.New("invalid services metadata format")
		}
	} else {
		return errors.New("no services metadata found in the bundle")
	}

	// Remove all modules related to the service
	b.bundle.Modules = slices.DeleteFunc(b.bundle.Modules, func(module opabundle.ModuleFile) bool {
		return strings.HasPrefix(module.Path, string(os.PathSeparator)+serviceName)
	})

	return nil
}
