package main

import (
	"dspn-regogenerator/cmd/cli/commands"
	"dspn-regogenerator/config"
	"dspn-regogenerator/internal/bundle"
	"fmt"
	"os"
	"path/filepath"

	opabundle "github.com/open-policy-agent/opa/v1/bundle"

	"github.com/spf13/cobra"
)

func main() {
	var serviceName, bucketName, minioServer, minioSecretKey, minioAccessKey string

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

	var localPath = ""

	var DeleteServicePolicyCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete policies related to a service",
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
			newBundle, err := bundle.RemoveService(b, serviceName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error removing service: %v\n", err)
				os.Exit(1)
			}
			files := bundle.ListBundleFiles(newBundle)
			services := map[string]struct{}{}
			fmt.Println("Remaining services")
			for _, f := range files {
				services[filepath.Dir(f)] = struct{}{}
			}
			for k := range services {
				fmt.Println(k)
			}
			err = bundle.WriteBundleToMinio(newBundle, config.Config.BundleFileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing bundle to Minio: %v\n", err)
				os.Exit(1)
			}
		},
	}
	DeleteServicePolicyCmd.Flags().StringVar(&serviceName, "service_name", "", "Name of the service (required)")
	DeleteServicePolicyCmd.MarkFlagRequired("service_name")
	DeleteServicePolicyCmd.Flags().StringVar(&localPath, "local-path", "", "")

	rootCmd.AddCommand(configCmd, DeleteServicePolicyCmd)

	rootCmd.AddCommand(commands.AddCmd, commands.ListCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
