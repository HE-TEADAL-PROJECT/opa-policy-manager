package usecases

import (
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"dspn-regogenerator/internal/generator"
	"dspn-regogenerator/internal/policy/parser"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func AddService(serviceName string, specData []byte) error {
	// Verify if the bundle exists on minio
	bundleExists, err := bundle.CheckBundleFileExists(config.LatestBundleName)
	if err != nil {
		return fmt.Errorf("error checking bundle existence: %v", err)
	}

	if !bundleExists {
		slog.Info(("No bundle found"))
		// Parse the OpenAPI spec to extract policies and provider
		policies, err := parser.ParseOpenAPIPolicies(specData)
		if err != nil || policies == nil {
			return fmt.Errorf("error parsing OpenAPI spec: %v", err)
		}
		provider, err := parser.ParseOpenAPIIAM(specData)
		if err != nil || provider == nil {
			return fmt.Errorf("error parsing OpenAPI provider: %v", err)
		}

		// Create a temporary directory for the output
		tempDir, err := os.MkdirTemp("", "bundle-*")
		if err != nil {
			return fmt.Errorf("error creating temp directory: %v", err)
		}
		regoDir := filepath.Join(tempDir, "rego")
		err = os.MkdirAll(regoDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating rego directory: %v", err)
		}

		// Generate the service folder
		err = generator.GenerateServiceFolder(serviceName, regoDir, *provider, policies)
		if err != nil {
			return fmt.Errorf("error generating service folder: %v", err)
		}

		// Generate the main.rego file
		err = generator.GenerateNewMain(tempDir, []string{serviceName})
		if err != nil {
			return fmt.Errorf("error generating main.rego: %v", err)
		}

		// Build the bundle
		b, err := bundle.BuildBundle(tempDir, "rego")
		if err != nil {
			return fmt.Errorf("error building bundle: %v", err)
		}
		if err := bundle.WriteBundleToMinio(b, config.LatestBundleName); err != nil {
			return fmt.Errorf("error writing bundle to Minio: %v", err)
		}
		slog.Info("Bundle created successfully and uploaded to Minio", "serviceName", serviceName)
	} else {
		// Load the existing bundle from Minio
		b, err := bundle.LoadBundleFromMinio(config.LatestBundleName)
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
		err = generator.GenerateServiceFolder(serviceName, regoDir, *provider, policies)
		if err != nil {
			return fmt.Errorf("error generating service folder: %v", err)
		}

		// Update the main.rego file
		err = generator.UpdateMainFile(tempDir, []string{serviceName})

		// Update the bundle with the new service
		newBundle, err := bundle.AddRegoFilesFromDirectory(b, tempDir)
		if err != nil {
			return fmt.Errorf("error adding rego files to bundle: %v", err)
		}

		// Copy the current bundle to a backup timestamped object
		newBundleName := config.TagBundleName(time.Now().Format("2006-01-02_15-04-05"))
		if err := bundle.RenameBundleFileName(config.LatestBundleName, newBundleName); err != nil {
			return fmt.Errorf("error renaming bundle file: %v", err)
		}

		// Write the updated bundle to Minio
		if err := bundle.WriteBundleToMinio(newBundle, config.LatestBundleName); err != nil {
			return fmt.Errorf("error writing updated bundle to Minio: %v", err)
		}
		slog.Info("Bundle updated successfully and uploaded to Minio", "serviceName", serviceName)
	}
	return nil
}
