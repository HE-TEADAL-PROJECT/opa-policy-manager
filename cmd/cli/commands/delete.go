package commands

import (
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"time"

	"github.com/spf13/cobra"
)

var ()

var DeleteCmd = &cobra.Command{
	Use:   "delete <service name>",
	Short: "Delete all policies related to a service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]
		// Check if the bundle exists on minio
		bundleExists, err := bundle.CheckBundleFileExists(config.LatestBundleName)
		if err != nil {
			cmd.PrintErrf("Error checking if service exists: %v\n", err)
			return
		}

		if !bundleExists {
			cmd.PrintErrf("Bundle does not exist. Cannot delete policies.\n")
			return
		}

		// Load the bundle from minio
		b, err := bundle.LoadBundleFromMinio(config.LatestBundleName)
		if err != nil {
			cmd.PrintErrf("Error loading bundle from Minio: %v\n", err)
			return
		}

		// Delete the service from the bundle
		newBundle, err := bundle.RemoveService(b, serviceName)
		if err != nil {
			cmd.PrintErrf("Error deleting policies for service %s: %v\n", serviceName, err)
			return
		}

		// Copy the current bundle to a backup timestamped object
		newBundleName := config.TagBundleName(time.Now().Format("2006-01-02_15-04-05"))
		if err := bundle.RenameBundleFileName(config.LatestBundleName, newBundleName); err != nil {
			cmd.PrintErrf("Error renaming bundle file: %v\n", err)
			return
		}

		// Save the new bundle to Minio
		if err := bundle.WriteBundleToMinio(newBundle, config.LatestBundleName); err != nil {
			cmd.PrintErrf("Error saving new bundle to Minio: %v\n", err)
			return
		}

		cmd.Printf("Successfully deleted policies for service %s and saved the new bundle to Minio.\n", serviceName)
	},
}
