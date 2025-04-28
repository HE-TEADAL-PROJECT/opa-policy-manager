package commands

import (
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"dspn-regogenerator/internal/generator"
	"dspn-regogenerator/internal/policy/parser"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	openAPISpec string
)

func loadSpecFile(specFile string) ([]byte, error) {
	// Load the OpenAPI spec file
	specData, err := os.ReadFile(specFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI spec file: %v", err)
	}
	return specData, nil
}

var AddCmd = &cobra.Command{
	Use:   "add [--spec <path/to/openapi/spec>] <service name>",
	Short: "Add policies related to a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]
		if openAPISpec == "" {
			cmd.Help()
			return
		}

		// Verify if the bundle exists on minio
		bundleExists, err := bundle.CheckBundleFileExists(config.LatestBundleName)
		if err != nil {
			cmd.PrintErrf("Error checking if service exists: %v\n", err)
			return
		}

		if !bundleExists {
			cmd.Printf("Bundle does not exist. Generating new bundle...\n")

			// Load the OpenAPI spec file
			specData, err := loadSpecFile(openAPISpec)
			if err != nil {
				cmd.PrintErrf("Error loading OpenAPI spec file: %v\n", err)
				return
			}

			// Parse the OpenAPI spec to extract policies and provider
			policies, err := parser.ParseOpenAPIPolicies(specData)
			if err != nil || policies == nil {
				cmd.PrintErrf("Error parsing OpenAPI spec: %v\n", err)
				return
			}
			provider, err := parser.ParseOpenAPIIAM(specData)
			if err != nil || provider == nil {
				cmd.PrintErrf("Error parsing OpenAPI provider: %v\n", err)
				return
			}

			// Create a temporary directory for the output
			tempDir, err := os.MkdirTemp("", "bundle-*")
			if err != nil {
				cmd.PrintErrf("Error creating temp directory: %v\n", err)
				return
			}
			regoDir := filepath.Join(tempDir, "rego")
			err = os.MkdirAll(regoDir, os.ModePerm)
			if err != nil {
				cmd.PrintErrf("Error creating rego directory: %v\n", err)
				return
			}

			// Generate the service folder
			err = generator.GenerateServiceFolder(serviceName, regoDir, *provider, policies)
			if err != nil {
				cmd.PrintErrf("Error generating service folder: %v\n", err)
				return
			}

			// Build the bundle
			b, err := bundle.BuildBundle(tempDir, "rego")
			if err != nil {
				cmd.PrintErrf("Error building bundle: %v\n", err)
				return
			}
			if err := bundle.WriteBundleToMinio(b, config.LatestBundleName); err != nil {
				cmd.PrintErrf("Error writing bundle to Minio: %v\n", err)
				return
			}
			cmd.Printf("Bundle created and uploaded successfully, a copy is available at %s\n", tempDir)
		} else {
			cmd.Printf("Bundle already exists. Loading existing bundle...\n")

			// Load the existing bundle from Minio
			b, err := bundle.LoadBundleFromMinio(config.LatestBundleName)
			if err != nil {
				cmd.PrintErrf("Error loading bundle from Minio: %v\n", err)
				return
			}

			// Create a temporary directory for the output
			tempDir, err := os.MkdirTemp("", "bundle-patch-*")
			if err != nil {
				cmd.PrintErrf("Error creating temp directory: %v\n", err)
				return
			}
			regoDir := filepath.Join(tempDir, "rego")
			err = os.MkdirAll(regoDir, os.ModePerm)
			if err != nil {
				cmd.PrintErrf("Error creating rego directory: %v\n", err)
				return
			}

			// Load the OpenAPI spec file
			specData, err := loadSpecFile(openAPISpec)
			if err != nil {
				cmd.PrintErrf("Error loading OpenAPI spec file: %v\n", err)
				return
			}

			// Parse the OpenAPI spec to extract policies and provider
			policies, err := parser.ParseOpenAPIPolicies(specData)
			if err != nil || policies == nil {
				cmd.PrintErrf("Error parsing OpenAPI spec: %v\n", err)
				return
			}
			provider, err := parser.ParseOpenAPIIAM(specData)
			if err != nil || provider == nil {
				cmd.PrintErrf("Error parsing OpenAPI provider: %v\n", err)
				return
			}

			// Generate the service folder
			err = generator.GenerateServiceFolder(serviceName, regoDir, *provider, policies)
			if err != nil {
				cmd.PrintErrf("Error generating service folder: %v\n", err)
				return
			}

			// Update the bundle with the new service
			newBundle, err := bundle.AddRegoFilesFromDirectory(b, tempDir)
			if err != nil {
				cmd.PrintErrf("Error adding rego files to bundle: %v\n", err)
				return
			}

			// Copy the current bundle to a backup timestamped object
			newBundleName := config.TagBundleName(time.Now().Format("2006-01-02_15-04-05"))
			if err := bundle.RenameBundleFileName(config.LatestBundleName, newBundleName); err != nil {
				cmd.PrintErrf("Error renaming bundle file: %v\n", err)
				return
			}

			// Write the updated bundle to Minio
			if err := bundle.WriteBundleToMinio(newBundle, config.LatestBundleName); err != nil {
				cmd.PrintErrf("Error writing bundle to Minio: %v\n", err)
				return
			}
			cmd.Printf("Bundle updated successfully and uploaded to Minio, new files are available at %s.\n", tempDir)
		}
	},
}

func init() {
	AddCmd.Flags().StringVar(&openAPISpec, "spec", "", "OpenAPI spec filename (required)")
	AddCmd.MarkFlagRequired("spec")
}
