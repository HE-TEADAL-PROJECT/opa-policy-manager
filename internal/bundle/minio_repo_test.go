package bundle

import (
	"bytes"
	"context"
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

func TestNewMinioRepository(t *testing.T) {
	ctx := context.Background()
	minioContainer := createMinioContainer(ctx, t)
	connectionString, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	repo, err := NewMinioRepository(
		connectionString,
		adminKey,
		adminSecret,
		false,
		"test-bucket",
	)
	if err != nil {
		t.Fatalf("Failed to create MinioRepository: %v", err)
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

func TestMinioRepository(t *testing.T) {
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

	repo, err := NewMinioRepository(
		connectionString,
		adminKey,
		adminSecret,
		false,
		"test-bucket",
	)
	if err != nil {
		t.Fatalf("Failed to create MinioRepository: %v", err)
	}

	opaBundle := prepareOpaBundle(t, []string{"service1"}, map[string]string{"/service1/service.rego": "package service1\ndefault allow = false"})
	bundle := &Bundle{
		bundle:       &opaBundle,
		serviceNames: []string{"service1"},
	}

	t.Run("WriteBundle", func(t *testing.T) {
		t.Cleanup(func() {
			if err := client.RemoveObject(ctx, "test-bucket", "test-bundle.tar.gz", miniosdk.RemoveObjectOptions{}); err != nil {
				t.Fatalf("Failed to remove object: %v", err)
			}
		})
		err := repo.Save("test-bundle.tar.gz", *bundle)
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
		bundle, err := repo.Get("test-bundle.tar.gz")
		if err != nil {
			t.Fatalf("Failed to read bundle: %v", err)
		}
		if bundle == nil {
			t.Fatalf("Expected non-nil bundle, got nil")
		}
		if len(bundle.serviceNames) != 1 {
			t.Fatalf("Expected 1 service, got %d", len(bundle.bundle.Manifest.Metadata["services"].([]string)))
		}
		if bundle.serviceNames[0] != "service1" {
			t.Fatalf("Expected service name 'service1', got '%s'", bundle.bundle.Manifest.Metadata["services"].([]string)[0])
		}
	})
}
