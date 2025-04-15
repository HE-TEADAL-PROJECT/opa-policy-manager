package main

import (
	"dspn-regogenerator/commands"
	"dspn-regogenerator/config"
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/generator"
	"dspn-regogenerator/internal/policy/parser"
	"fmt"
	"os"
	"path/filepath"

	opabundle "github.com/open-policy-agent/opa/v1/bundle"

	"github.com/spf13/cobra"
)

func main() {
	var serviceName, openAPISpec, bucketName, minioServer, minioSecretKey, minioAccessKey string

	var rootCmd = &cobra.Command{Use: "dspn-regogenerator"}

	config.LoadConfigFromFile()

	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Configure the DSPN manager",
		Run: func(cmd *cobra.Command, args []string) {
			config.Config.Minio_Server = minioServer
			config.Config.Minio_Access_Key = minioAccessKey
			config.Config.Minio_Secret_Key = minioSecretKey
			config.Config.Bucket_Name = bucketName
			config.Config.BundleName = "teadal-policy-bundle-LATEST"
			config.Config.BundleFileName = "teadal-policy-bundle-LATEST.tar.gz"
			err := config.TestMinio()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cannot connect to the Minio server ")
			} else {
				config.SaveConfigToFile()
			}

		},
	}
	configCmd.Flags().StringVar(&minioServer, "minio_server", "", "Minio server URL")
	configCmd.Flags().StringVar(&minioAccessKey, "minio_access_key", "", "Minio access key")
	configCmd.Flags().StringVar(&minioSecretKey, "minio_secret_key", "", "Minio secret key")
	configCmd.Flags().StringVar(&bucketName, "bucket_name", "opa-policy-bundles", "Bucket name (Optional)")
	configCmd.MarkFlagRequired("minio_server")
	configCmd.MarkFlagRequired("minio_access_key")
	configCmd.MarkFlagRequired("minio_secret_key")

	var addServicePolicyCmd = &cobra.Command{
		Use:   "add",
		Short: "Add policies related to a service",
		Run: func(cmd *cobra.Command, args []string) {
			exists, err := bundle.CheckBundleFileExists(config.Config.BundleFileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error checking if service exists: %v\n", err)
				os.Exit(1)
			}
			if !exists {
				outputDir, _ := filepath.Abs("./output")
				bundleDir, _ := os.MkdirTemp(outputDir, "bundle*")
				mainDir := "rego"
				regoOutput := filepath.Join(bundleDir, mainDir)
				spec, err := loadBundle(openAPISpec)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error loading bundle: %v\n", err)
					os.Exit(1)
				}
				policies, err := parser.ParseOpenAPIPolicies(spec)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing OpenAPI policies: %v\n", err)
					os.Exit(1)
				}
				IAMprovider, err := parser.ParseOpenAPIIAM(spec)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing OpenAPI IAM: %v\n", err)
					os.Exit(1)
				}
				err = generator.GenerateServiceFolder(serviceName, regoOutput, *IAMprovider, policies)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error generating service folder: %v\n", err)
					os.Exit(1)
				}
				generator.GenerateStaticFolders(regoOutput)
				fmt.Printf("Service folder generated successfully at %s\n", outputDir)
				b, err := bundle.BuildBundle(bundleDir, mainDir)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error building bundle: %v\n", err)
					os.Exit(1)
				}
				err = bundle.WriteBundleToMinio(b, config.Config.BundleFileName)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error writing bundle to minio: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Println("Service already exists in the bundle.")
			}
		},
	}
	addServicePolicyCmd.Flags().StringVar(&serviceName, "service_name", "", "Name of the service (required)")
	addServicePolicyCmd.Flags().StringVar(&openAPISpec, "openAPIspec", "", "OpenAPI spec filename (required)")
	addServicePolicyCmd.MarkFlagRequired("service_name")
	addServicePolicyCmd.MarkFlagRequired("openAPIspec")

	var localPath = ""
	var ListServicePoliciesCmd = &cobra.Command{
		Use:   "list",
		Short: "Add policies related to a service",
		Run: func(cmd *cobra.Command, args []string) {
			var b = new(opabundle.Bundle)
			var err error
			if localPath != "" {
				b, err = bundle.LoadBundleFromFile(localPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error loading bundle from local path: %v\n", err)
					os.Exit(1)
				}
			} else {
				b, err = bundle.LoadBundleFromMinio(config.Config.BundleFileName)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error loading bundle from Minio: %v\n", err)
					os.Exit(1)
				}
			}
			files := bundle.ListBundleFiles(b)
			services := map[string]struct{}{}
			fmt.Println("services available")
			for _, f := range files {
				services[filepath.Dir(f)] = struct{}{}
			}
			for k := range services {
				fmt.Println(k)
			}
		},
	}
	ListServicePoliciesCmd.Flags().StringVar(&localPath, "local-path", "", "")

	var DeleteServicePolicyCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete policies related to a service",
		Run: func(cmd *cobra.Command, args []string) {
			commands.DeleteServicePolicies(serviceName)

		},
	}
	DeleteServicePolicyCmd.Flags().StringVar(&serviceName, "service_name", "", "Name of the service (required)")
	DeleteServicePolicyCmd.MarkFlagRequired("service_name")

	rootCmd.AddCommand(configCmd, addServicePolicyCmd, ListServicePoliciesCmd, DeleteServicePolicyCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
