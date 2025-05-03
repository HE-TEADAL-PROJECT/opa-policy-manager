package usecases

import (
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"fmt"
	"log/slog"
	"time"
)

func DeleteService(serviceName string) error {
	// Check if the bundle exists on minio
	bundleExists, err := bundle.CheckBundleFileExists(config.LatestBundleName)
	if err != nil {
		return fmt.Errorf("error checking bundle existence: %v", err)
	}

	if !bundleExists {
		return fmt.Errorf("bundle %s not found", config.LatestBundleName)
	}

	// Load the bundle from minio
	b, err := bundle.LoadBundleFromMinio(config.LatestBundleName)
	if err != nil {
		return fmt.Errorf("error loading bundle from Minio: %v", err)
	}

	// Delete the service from the bundle
	newBundle, err := bundle.RemoveService(b, serviceName)
	if err != nil {
		return fmt.Errorf("error deleting policies for service %s: %v", serviceName, err)
	}

	// Copy the current bundle to a backup timestamped object
	newBundleName := config.TagBundleName(time.Now().Format("2006-01-02_15-04-05"))
	if err := bundle.RenameBundleFileName(config.LatestBundleName, newBundleName); err != nil {
		return fmt.Errorf("error renaming bundle file: %v", err)
	}

	// Save the new bundle to Minio
	if err := bundle.WriteBundleToMinio(newBundle, config.LatestBundleName); err != nil {
		return fmt.Errorf("error saving new bundle to Minio: %v", err)
	}

	slog.Info("Successfully deleted policies for service", "service", serviceName)
	return nil
}
