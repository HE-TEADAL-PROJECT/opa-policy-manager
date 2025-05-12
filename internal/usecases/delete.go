package usecases

import (
	"context"
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"fmt"
	"log/slog"
	"time"
)

func DeleteService(serviceName string) error {
	minioRepo, err := bundle.NewMinioRepositoryFromConfig()
	if err != nil {
		return fmt.Errorf("error creating minio repository: %v", err)
	}
	ctx := context.Background()

	// Check if the bundle exists on minio
	bundleExists, err := minioRepo.BundleExists(ctx, config.LatestBundleName)
	if err != nil {
		return fmt.Errorf("error checking bundle existence: %v", err)
	}

	if !bundleExists {
		return fmt.Errorf("bundle %s not found", config.LatestBundleName)
	}

	// Load the bundle from minio
	b, err := minioRepo.Read(config.LatestBundleName)
	if err != nil {
		return fmt.Errorf("error loading bundle from Minio: %v", err)
	}

	// Delete the service from the bundle
	if err := b.RemoveService(serviceName); err != nil {
		return fmt.Errorf("error deleting policies for service %s: %v", serviceName, err)
	}

	// Copy the current bundle to a backup timestamped object
	newBundleName := config.TagBundleName(time.Now().Format("2006-01-02_15-04-05"))
	if err := minioRepo.CopyBundle(ctx, config.LatestBundleName, newBundleName); err != nil {
		return fmt.Errorf("error renaming bundle file: %v", err)
	}

	// Save the new bundle to Minio
	if err := minioRepo.Write(config.LatestBundleName, *b); err != nil {
		return fmt.Errorf("error saving new bundle to Minio: %v", err)
	}

	slog.Info("Successfully deleted policies for service", "service", serviceName)
	return nil
}
