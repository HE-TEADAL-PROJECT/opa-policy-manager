package main

import (
	"context"
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"dspn-regogenerator/internal/minioutil"
	"log/slog"
	"os"
)

func main() {
	ctx := context.Background()

	minioClient, err := minioutil.NewFromConfig()

	if err != nil {
		slog.Error("failed to create minio client", "error", err)
		os.Exit(1)
	}

	_, err = minioClient.ListBuckets(ctx)
	if err != nil {
		slog.Error("Failed to connect minio client", "error", err)
		os.Exit(1)
	}

	err = minioutil.EnsureBucket(ctx, minioClient, config.MinioBucket)
	if err != nil {
		slog.Error("failed to ensure minio bucket", "error", err)
		os.Exit(1)
	}
	slog.Info("minio bucket is ready")

	repo := bundle.NewMinioRepositoryFromClient(minioClient, config.MinioBucket)

	server := createServer(repo)
	err = server.ListenAndServe()
	if err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
