package bundle

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	opabundle "github.com/open-policy-agent/opa/v1/bundle"
	"github.com/testcontainers/testcontainers-go/modules/minio"
)

const adminKey = "admin"
const adminSecret = "adminadmin"

func createMinioContainer(ctx context.Context, t *testing.T) *minio.MinioContainer {
	minioContainer, err := minio.Run(ctx, "minio/minio:latest", minio.WithUsername(adminKey), minio.WithPassword(adminSecret))
	if err != nil {
		t.Fatalf("Failed to start Minio container: %v", err)
	}
	t.Cleanup(func() {
		if err := minioContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})
	return minioContainer
}

func TestNewMinioBundleRepository(t *testing.T) {
	ctx := context.Background()
	minioContainer := createMinioContainer(ctx, t)
	connectionString, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	repo, err := NewMinioBundleRepository(
		connectionString,
		adminKey,
		adminSecret,
		false,
		"test-bucket",
	)
	if err != nil {
		t.Fatalf("Failed to create MinioBundleRepository: %v", err)
	}

	if repo == nil {
		t.Fatalf("Expected non-nil repository, got nil")
	}
	if repo.client == nil {
		t.Fatalf("Expected non-nil client, got nil")
	}
	if repo.bucket != "test-bucket" {
		t.Fatalf("Expected bucket name 'test-bucket', got '%s'", repo.bucket)
	}
	if _, err := repo.client.HealthCheck(5 * time.Second); err != nil {
		t.Fatalf("Expected client to be healthy, but it is not")
	}
}

func createBundleFromFiles(t *testing.T, files map[string]string, servicesName []string) *Bundle {
	// Create a bundle from files or other sources
	// This is a placeholder function. Replace with actual bundle creation logic.
	tempDir := t.TempDir()
	for name, content := range files {
		fullPath := filepath.Join(tempDir, name)
		fullDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(fullDir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}
	if opa, err := opabundle.NewCustomReader(opabundle.NewFSLoaderWithRoot(os.DirFS(tempDir), ".")).Read(); err != nil {
		t.Fatalf("Failed to read bundle: %v", err)
	} else {
		bundle := &Bundle{
			bundle: &opa,
		}
		bundle.bundle.Manifest.Metadata = make(map[string]interface{})
		bundle.bundle.Manifest.Metadata["services"] = servicesName
		return bundle
	}
	return nil
}

func TestMinioBundleRepository(t *testing.T) {
	ctx := context.Background()
	minioContainer := createMinioContainer(ctx, t)
	connectionString, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	client, err := miniosdk.New(
		connectionString, &miniosdk.Options{Creds: credentials.NewStaticV4(adminKey, adminSecret, ""), Secure: false},
	)
	if err != nil {
		t.Fatalf("Failed to create Minio client: %v", err)
	}
	client.MakeBucket(ctx, "test-bucket", miniosdk.MakeBucketOptions{})

	repo, err := NewMinioBundleRepository(
		connectionString,
		adminKey,
		adminSecret,
		false,
		"test-bucket",
	)
	if err != nil {
		t.Fatalf("Failed to create MinioBundleRepository: %v", err)
	}

	bundle := createBundleFromFiles(t, map[string]string{"service1/service.rego": "package service1\ndefault allow = false"}, []string{"service1"})

	t.Run("WriteBundle", func(t *testing.T) {
		t.Cleanup(func() {
			if err := client.RemoveObject(ctx, "test-bucket", "test-bundle.tar.gz", miniosdk.RemoveObjectOptions{}); err != nil {
				t.Fatalf("Failed to remove object: %v", err)
			}
		})
		err := repo.Write("test-bundle.tar.gz", *bundle)
		if err != nil {
			t.Fatalf("Failed to write bundle: %v", err)
		}
		reader, err := client.GetObject(ctx, "test-bucket", "test-bundle.tar.gz", miniosdk.GetObjectOptions{})
		if err != nil {
			t.Fatalf("Failed to get object: %v", err)
		}
		defer reader.Close()
	})

	buffer := &bytes.Buffer{}

	t.Run("ReadBundle", func(t *testing.T) {
		// Push the bundle to MinIO
		if err := opabundle.NewWriter(buffer).Write(*bundle.bundle); err != nil {
			t.Fatalf("Failed to write bundle: %v", err)
		}
		if _, err := client.PutObject(ctx, "test-bucket", "test-bundle.tar.gz", buffer, int64(buffer.Len()), miniosdk.PutObjectOptions{}); err != nil {
			t.Fatalf("Failed to put object: %v", err)
		}

		// Read the bundle from MinIO
		bundle, err := repo.Read("test-bundle.tar.gz")
		if err != nil {
			t.Fatalf("Failed to read bundle: %v", err)
		}
		if bundle == nil {
			t.Fatalf("Expected non-nil bundle, got nil")
		}
		if len(bundle.bundle.Manifest.Metadata["services"].([]string)) != 1 {
			t.Fatalf("Expected 1 service, got %d", len(bundle.bundle.Manifest.Metadata["services"].([]string)))
		}
		if bundle.bundle.Manifest.Metadata["services"].([]string)[0] != "service1" {
			t.Fatalf("Expected service name 'service1', got '%s'", bundle.bundle.Manifest.Metadata["services"].([]string)[0])
		}
	})
}
