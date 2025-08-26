package commands

import (
	"context"
	"dspn-regogenerator/internal/config"
	"dspn-regogenerator/internal/usecases"

	"github.com/spf13/cobra"
)

var TestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the bundle",
	Long:  `Test the bundle against the httpbin service. This will run all the tests in the bundle and print the results.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		// Print the current configuration
		cmd.Println("Configuration:")
		cmd.Println("    MinIO Endpoint:", config.MinioEndpoint)
		cmd.Println("    MinIO Access Key:", config.MinioAccessKey)
		cmd.Println("    MinIO Secret Key:", config.MinioSecretKey)
		cmd.Println("    MinIO Bucket:", config.MinioBucket)
		cmd.Println("    MinIO Bundle Prefix:", config.MinioBundlePrefix)
		cmd.Println("    MinIO Timeout:", config.MinioTimeout)

		if err := usecases.InitialTest(ctx); err != nil {
			cmd.PrintErrf("Error during initial test: %v\n", err)
		}
	},
}
