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

package usecases

import (
	"context"
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"fmt"
)

// Describes the structure of the bundle, in terms of which files and services are available
type BundleStructure struct {
	Files    []string `json:"files"`
	Services []string `json:"dirs"`
}

// Implement the usecase to get the bundle structure from minio, inspecting it and returning the structure
func GetBundleStructure() (*BundleStructure, error) {
	minioRepo, err := bundle.NewMinioRepositoryFromConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating minio repository: %v", err)
	}
	ctx := context.Background()
	bundleExists, err := minioRepo.BundleExists(ctx, config.LatestBundleName)
	if err != nil {
		return nil, err
	}
	if !bundleExists {
		return nil, fmt.Errorf("bundle %s not found", config.LatestBundleName)
	}

	loadedBundle, err := minioRepo.Read(config.LatestBundleName)
	if err != nil {
		return nil, err
	}

	services, err := loadedBundle.Services()
	if err != nil {
		return nil, fmt.Errorf("error getting services from bundle: %v", err)
	}

	bundleStructure := &BundleStructure{
		// TODO: fix this, we need to get the files from the bundle
		Files:    []string{},
		Services: services,
	}

	return bundleStructure, nil
}
