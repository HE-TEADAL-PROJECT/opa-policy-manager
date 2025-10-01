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
	"dspn-regogenerator/internal/generator"
	"dspn-regogenerator/internal/policy/parser"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"
)

func AddService(serviceName string, specData []byte) error {
	minioRepo, err := bundle.NewMinioRepositoryFromConfig()
	if err != nil {
		return fmt.Errorf("error creating minio repository: %v", err)
	}
	ctx := context.Background()

	// Verify if the bundle exists on minio
	bundleExists, err := minioRepo.BundleExists(ctx, config.LatestBundleName)
	if err != nil {
		return fmt.Errorf("error checking bundle existence: %v", err)
	}
	if !bundleExists {
		return fmt.Errorf("bundle %s does not exist in Minio", config.LatestBundleName)
	}

	// Load the existing bundle from Minio
	b, err := minioRepo.Read(config.LatestBundleName)
	if err != nil {
		return fmt.Errorf("error loading bundle from Minio: %v", err)
	}

	// Create a temporary directory for the output
	tempDir, err := os.MkdirTemp("", "bundle-patch-*")
	if err != nil {
		return fmt.Errorf("error creating temp directory: %v", err)
	}
	regoDir := filepath.Join(tempDir, "rego")
	err = os.MkdirAll(regoDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating rego directory: %v", err)
	}

	// Parse the OpenAPI spec to extract policies and provider
	policies, err := parser.ParseOpenAPIPolicies(specData)
	if err != nil || policies == nil {
		return fmt.Errorf("error parsing OpenAPI spec: %v", err)
	}
	provider, err := parser.ParseOpenAPIIAM(specData)
	if err != nil || provider == nil {
		return fmt.Errorf("error parsing OpenAPI provider: %v", err)
	}

	// Generate the service folder
	options := generator.ServiceOptions{
		ServiceName: serviceName,
		PathPrefix:  "/" + serviceName,
	}
	err = generator.GenerateServiceFolder(options, regoDir, *provider, policies)
	if err != nil {
		return fmt.Errorf("error generating service folder: %v", err)
	}

	// Load the regoDir folder and compose a map[string][]byte
	regoFiles := make(map[string][]byte)
	err = filepath.Walk(regoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relativePath, err := filepath.Rel(tempDir, path)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			regoFiles[string(os.PathSeparator)+filepath.ToSlash(relativePath)] = content
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error reading rego files: %v", err)
	}
	fmt.Print("regoFiles: ", slices.Collect(maps.Keys(regoFiles)), "\n")

	err = b.AddService(serviceName, regoFiles)
	if err != nil {
		return fmt.Errorf("error adding service to bundle: %v", err)
	}

	services, err := b.Services()
	if err != nil {
		return fmt.Errorf("error getting services from bundle: %v", err)
	}
	if err := generator.GenerateNewMain(regoDir, services); err != nil {
		return fmt.Errorf("error generating main.rego: %v", err)
	}
	if err := b.LoadNewMain(filepath.Join(regoDir, "main.rego")); err != nil {
		return fmt.Errorf("error loading new main.rego: %v", err)
	}

	// Copy the current bundle to a backup timestamped object
	newBundleName := config.TagBundleName(time.Now().Format("2006-01-02_15-04-05"))
	if err := minioRepo.CopyBundle(ctx, config.LatestBundleName, newBundleName); err != nil {
		return fmt.Errorf("error renaming bundle file: %v", err)
	}

	// Write the updated bundle to Minio
	if err := minioRepo.Write(config.LatestBundleName, *b); err != nil {
		return fmt.Errorf("error writing updated bundle to Minio: %v", err)
	}
	slog.Info("Bundle updated successfully and uploaded to Minio", "serviceName", serviceName)
	return nil
}
