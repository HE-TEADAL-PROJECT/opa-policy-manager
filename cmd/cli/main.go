package main

import (
	"dspn-regogenerator/cmd/cli/commands"
	"dspn-regogenerator/config"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var bucketName, minioServer, minioSecretKey, minioAccessKey string

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

	rootCmd.AddCommand(configCmd)

	rootCmd.AddCommand(commands.AddCmd, commands.ListCmd, commands.DeleteCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
