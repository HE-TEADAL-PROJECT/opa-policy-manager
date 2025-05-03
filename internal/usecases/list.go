package usecases

import (
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
)

// Describes the structure of the bundle, in terms of which files and services are available
type BundleStructure struct {
	Files    []string `json:"files"`
	Services []string `json:"dirs"`
}

// Implement the usecase to get the bundle structure from minio, inspecting it and returning the structure
func GetBundleStructure() (*BundleStructure, error) {
	bundleExists, err := bundle.CheckBundleFileExists(config.LatestBundleName)
	if err != nil {
		return nil, err
	}
	if !bundleExists {
		return nil, fmt.Errorf("bundle %s not found", config.LatestBundleName)
	}

	loadedBundle, err := bundle.LoadBundleFromMinio(config.LatestBundleName)
	if err != nil {
		return nil, err
	}

	files := bundle.ListBundleFiles(loadedBundle)
	dirs := map[string][]string{}
	for _, dir := range files {
		dirs[filepath.Dir(dir)] = append(dirs[filepath.Dir(dir)], filepath.Base(dir))
	}

	bundleStructure := &BundleStructure{
		Files:    files,
		Services: slices.Collect(maps.Keys(dirs)),
	}

	return bundleStructure, nil
}
