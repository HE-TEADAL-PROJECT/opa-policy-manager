package bundle_test

import (
	"context"
	"dspn-regogenerator/config"
	"dspn-regogenerator/internal/bundle"
	"os"
	"path/filepath"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	opabundle "github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/compile"
)

func TestBuildBundle(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a mock bundle directory with a sample policy file
	bundleDir := filepath.Join(tempDir, "bundle")
	err := os.Mkdir(bundleDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create bundle directory: %v", err)
	}

	policyFile := filepath.Join(bundleDir, "example.rego")
	err = os.WriteFile(policyFile, []byte(`package example

	default allow = false`), 0644)
	if err != nil {
		t.Fatalf("Failed to create policy file: %v", err)
	}

	// Call BuildBundle
	b, err := bundle.BuildBundle(bundleDir, tempDir)
	if err != nil {
		t.Fatalf("Failed to build bundle: %v", err)
	}
	// Verify the bundle is not nil
	if b == nil {
		t.Fatal("Expected a non-nil bundle")
	}
}

func TestWriteBundleToFile(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a mock bundle
	b := &opabundle.Bundle{}

	// Call WriteBundleToFile
	err := bundle.WriteBundleToFile(b, tempDir+"/bundle.tar.gz")
	if err != nil {
		t.Fatalf("Failed to write bundle to file: %v", err)
	}

	// Verify the file exists
	filePath := filepath.Join(tempDir, "bundle.tar.gz")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected bundle file to exist: %v", err)
	}
}

func TestLoadBundleFromFile(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a mock bundle
	os.WriteFile(tempDir+"/example.rego", []byte(`package example
default allow = false
`), 0644)

	// Create a mock bundle file
	bundleFilePath := filepath.Join(tempDir, "bundle.tar.gz")
	bundleFile, err := os.Create(bundleFilePath)
	if err != nil {
		t.Fatalf("Failed to create bundle file: %v", err)
	}
	defer bundleFile.Close()

	compile.New().WithAsBundle(true).WithPaths(tempDir).WithOutput(bundleFile).Build(context.TODO())

	// Call LoadBundleFromFile
	b, err := bundle.LoadBundleFromFile(bundleFilePath)
	if err != nil {
		t.Fatalf("Failed to load bundle from file: %v", err)
	}

	// Verify the bundle is not nil
	if b == nil {
		t.Fatal("Expected a non-nil bundle")
	}
	if len(b.Modules) != 1 {
		t.Fatal("Expected bundle to contain 1 module")
	}
	if b.Modules[0].Parsed.Package.String() != "package example" {
		t.Fatalf("Expected module to be 'package example', got '%s'", b.Modules[0].Parsed.Package.String())
	}
}
func TestWriteBundleToMinio(t *testing.T) {
	// Mock the MinIO client
	mockMinioServer := "localhost:9000"
	mockAccessKey := "admin"
	mockSecretKey := "adminadmin"
	mockBucketName := "test-bucket"

	// Set up the mock configuration
	config.Config.Minio_Server = mockMinioServer
	config.Config.Minio_Access_Key = mockAccessKey
	config.Config.Minio_Secret_Key = mockSecretKey
	config.Config.Bucket_Name = mockBucketName
	config.Config.BundleFileName = "bundle.tar.gz"

	// Create a mock MinIO server (use a library like minio/minio-go-mock or similar for real tests)
	// For simplicity, this test assumes the MinIO server is running locally.

	// Create a mock bundle
	b := &opabundle.Bundle{
		Modules: []opabundle.ModuleFile{
			{
				Path:   "example.rego",
				Raw:    []byte(`package example; default allow = false`),
				Parsed: nil,
			},
		},
	}

	// Call WriteBundleToMinio
	err := bundle.WriteBundleToMinio(b)
	if err != nil {
		t.Fatalf("Failed to write bundle to MinIO: %v", err)
	}

	client, err := minio.New(config.Config.Minio_Server, &minio.Options{
		Creds: credentials.NewStaticV4(config.Config.Minio_Access_Key, config.Config.Minio_Secret_Key, ""),
	})
	if err != nil {
		t.Fatalf("Failed to create MinIO client: %v", err)
	}
	// Verify the bucket exists
	exists, err := client.BucketExists(context.Background(), config.Config.Bucket_Name)
	if err != nil {
		t.Fatalf("Failed to check if bucket exists: %v", err)
	}
	if !exists {
		t.Fatalf("Expected bucket to exist, but it does not")
	}

	defer func() {
		// Clean up the object after the test
		err := client.RemoveObject(context.Background(), config.Config.Bucket_Name, "bundle.tar.gz", minio.RemoveObjectOptions{})
		if err != nil {
			t.Fatalf("Failed to remove object: %v", err)
		}
		// Clean up the bucket after the test
		err = client.RemoveBucket(context.Background(), config.Config.Bucket_Name)
		if err != nil {
			t.Fatalf("Failed to remove bucket: %v", err)
		}
	}()

	// Verify the object exists
	objectName := "bundle.tar.gz"
	objectInfo, err := client.StatObject(context.Background(), config.Config.Bucket_Name, objectName, minio.StatObjectOptions{})
	if err != nil {
		t.Fatalf("Failed to stat object: %v", err)
	}
	if objectInfo.Key != objectName {
		t.Fatalf("Expected object name to be '%s', got '%s'", objectName, objectInfo.Key)
	}
}
func TestLoadBundleFromMinio(t *testing.T) {
	// Mock the MinIO client
	mockMinioServer := "localhost:9000"
	mockAccessKey := "admin"
	mockSecretKey := "adminadmin"
	mockBucketName := "test-bucket"

	// Set up the mock configuration
	config.Config.Minio_Server = mockMinioServer
	config.Config.Minio_Access_Key = mockAccessKey
	config.Config.Minio_Secret_Key = mockSecretKey
	config.Config.Bucket_Name = mockBucketName
	config.Config.BundleFileName = "bundle.tar.gz"

	client, err := minio.New(config.Config.Minio_Server, &minio.Options{
		Creds: credentials.NewStaticV4(config.Config.Minio_Access_Key, config.Config.Minio_Secret_Key, ""),
	})
	if err != nil {
		t.Fatalf("Failed to create MinIO client: %v", err)
	}

	// Create the bucket
	err = client.MakeBucket(context.Background(), config.Config.Bucket_Name, minio.MakeBucketOptions{})
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}
	defer func() {
		// Clean up the bucket after the test
		err = client.RemoveBucket(context.Background(), config.Config.Bucket_Name)
		if err != nil {
			t.Fatalf("Failed to remove bucket: %v", err)
		}
	}()

	// Create a mock bundle file
	tempDir := t.TempDir()
	os.WriteFile(tempDir+"/example.rego", []byte(`package example
default allow = false
`), 0644)
	bundleName := config.Config.BundleFileName
	file, err := os.CreateTemp(tempDir, bundleName)
	if err != nil {
		t.Fatalf("Failed to create bundle file: %v", err)
	}
	defer file.Close()
	compile.New().WithAsBundle(true).WithPaths(tempDir).WithOutput(file).Build(context.TODO())

	_, err = client.FPutObject(context.Background(), config.Config.Bucket_Name, config.Config.BundleFileName, file.Name(), minio.PutObjectOptions{})
	if err != nil {
		t.Fatalf("Failed to upload mock bundle to MinIO: %v", err)
	}
	defer func() {
		// Clean up the object after the test
		err := client.RemoveObject(context.Background(), config.Config.Bucket_Name, bundleName, minio.RemoveObjectOptions{})
		if err != nil {
			t.Fatalf("Failed to remove object: %v", err)
		}
	}()

	// Call LoadBundleFromMinio
	b, err := bundle.LoadBundleFromMinio()
	if err != nil {
		t.Fatalf("Failed to load bundle from MinIO: %v", err)
	}

	// Verify the bundle is not nil
	if b == nil {
		t.Fatal("Expected a non-nil bundle")
	}
	if len(b.Modules) != 1 {
		t.Fatalf("Expected bundle to contain 1 module, got %d", len(b.Modules))
	}
}
