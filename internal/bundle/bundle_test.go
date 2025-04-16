package bundle_test

import (
	"context"
	"dspn-regogenerator/config"
	"dspn-regogenerator/internal/bundle"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	opabundle "github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/compile"
)

func TestBuildBundle(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	mainDir := "rego"

	// Create a mock bundle directory with a sample policy file
	bundleDir := filepath.Join(tempDir, mainDir)
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
	b, err := bundle.BuildBundle(tempDir, mainDir)
	if err != nil {
		t.Fatalf("Failed to build bundle: %v", err)
	}
	// Verify the bundle is not nil
	if b == nil {
		t.Fatal("Expected a non-nil bundle")
	}
	if len(b.Modules) != 1 {
		t.Fatalf("Expected bundle to contain 1 module, got %d", len(b.Modules))
	}
	if b.Modules[0].Path != "rego/example.rego" {
		t.Fatalf("Expected module path to be 'rego/example.rego', got '%s'", b.Modules[0].Path)
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
	err := bundle.WriteBundleToMinio(b, config.Config.BundleFileName)
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
	b, err := bundle.LoadBundleFromMinio(config.Config.BundleFileName)
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

func TestListBundleDirectories(t *testing.T) {
	// Create rego directory in temp
	tempDir := t.TempDir()
	mainDir := "rego"
	if err := os.Mkdir(filepath.Join(tempDir, mainDir), 0755); err != nil {
		t.Fatalf("Failed to create rego directory: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tempDir, mainDir, "serviceA"), 0755); err != nil {
		t.Fatalf("Failed to create rego/serviceA")
	}

	// Create example rego file
	mainFile := filepath.Join(tempDir, mainDir, "main.rego")
	policyFile := filepath.Join(tempDir, mainDir, "serviceA", "example.rego")
	if err := os.WriteFile(mainFile, []byte(`package main
default allow = false`), 0644); err != nil {
		t.Fatalf("Failed to create policy file: %v", err)
	}
	if err := os.WriteFile(policyFile, []byte(`package example
default allow = false`), 0644); err != nil {
		t.Fatalf("Failed to create policy file: %v", err)
	}

	b, err := bundle.BuildBundle(tempDir, mainDir)
	if err != nil {
		t.Fatalf("Failed to build bundle: %v", err)
	}

	dirs := bundle.ListBundleFiles(b)
	if len(dirs) != 2 {
		t.Errorf("Expected only 2 element in the list, got %d", len(dirs))
	}
	if !slices.Contains(dirs, "main.rego") || !slices.Contains(dirs, "serviceA/example.rego") {
		t.Errorf("Expected to contain \"main.rego\" and \"serviceA/example.rego\", got %v", dirs)
	}
}

func TestAddRegoFilesFromDirectory(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a mock bundle
	originalBundle := opabundle.Bundle{
		Modules: []opabundle.ModuleFile{},
		Manifest: opabundle.Manifest{
			Metadata: map[string]interface{}{
				"main": "rego",
			},
		},
	}

	// Create a directory structure with .rego files
	regoDir := filepath.Join(tempDir, "rego", "serviceA")
	err := os.MkdirAll(regoDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create rego directory: %v", err)
	}

	// Create sample .rego files
	file1 := filepath.Join(regoDir, "policy1.rego")
	err = os.WriteFile(file1, []byte(`package policy1

	default allow = false`), 0644)
	if err != nil {
		t.Fatalf("Failed to create policy1.rego: %v", err)
	}

	file2 := filepath.Join(regoDir, "policy2.rego")
	err = os.WriteFile(file2, []byte(`package policy2

	default allow = true`), 0644)
	if err != nil {
		t.Fatalf("Failed to create policy2.rego: %v", err)
	}

	// Call AddRegoFilesFromDirectory
	newBundle, err := bundle.AddRegoFilesFromDirectory(&originalBundle, tempDir)
	if err != nil {
		t.Fatalf("Failed to add Rego files from directory: %v", err)
	}

	// Verify the new bundle contains the added modules
	if len(newBundle.Modules) != 2 {
		t.Fatalf("Expected 2 modules in the bundle, got %d", len(newBundle.Modules))
	}

	// Verify the paths of the added modules
	expectedPaths := []string{"serviceA/policy1.rego", "serviceA/policy2.rego"}
	for _, module := range bundle.ListBundleFiles(newBundle) {
		if !slices.Contains(expectedPaths, module) {
			t.Errorf("Unexpected module path: %s", module)
		}
	}

	// Verify the content of the added modules
	if string(newBundle.Modules[0].Raw) != `package policy1

	default allow = false` && string(newBundle.Modules[1].Raw) != `package policy2

	default allow = true` {
		t.Errorf("Module content does not match expected content")
	}
}

func TestAddRegoFilesFromDirectoryWithInvalidFile(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a mock bundle
	originalBundle := opabundle.Bundle{
		Modules: []opabundle.ModuleFile{},
	}

	// Create a directory structure with a valid and an invalid .rego file
	regoDir := filepath.Join(tempDir, "rego")
	err := os.Mkdir(regoDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create rego directory: %v", err)
	}

	// Create a valid .rego file
	validFile := filepath.Join(regoDir, "valid.rego")
	err = os.WriteFile(validFile, []byte(`package valid

	default allow = true`), 0644)
	if err != nil {
		t.Fatalf("Failed to create valid.rego: %v", err)
	}

	// Create an invalid .rego file
	invalidFile := filepath.Join(regoDir, "invalid.rego")
	err = os.WriteFile(invalidFile, []byte(`package invalid

	default allow =`), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid.rego: %v", err)
	}

	// Call AddRegoFilesFromDirectory
	_, err = bundle.AddRegoFilesFromDirectory(&originalBundle, regoDir)
	if err == nil {
		t.Fatal("Expected an error due to invalid .rego file, but got none")
	}
	if !strings.Contains(err.Error(), "failed to parse module") {
		t.Fatalf("Expected parse error, got: %v", err)
	}
}

func TestAddRegoFileFromDirectoryWithOverwrite(t *testing.T) {
	tempDir := t.TempDir()

	// Create a mock bundle
	originalBundle := opabundle.Bundle{
		Modules: []opabundle.ModuleFile{},
		Manifest: opabundle.Manifest{
			Metadata: map[string]interface{}{
				"main": "rego",
			},
		},
	}

	// Create a directory structure with .rego files
	regoDir := filepath.Join(tempDir, "rego", "serviceA")
	err := os.MkdirAll(regoDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create rego directory: %v", err)
	}

	// Create sample .rego files
	file1 := filepath.Join(regoDir, "policy1.rego")
	err = os.WriteFile(file1, []byte(`package policy1

	default allow = false`), 0644)
	if err != nil {
		t.Fatalf("Failed to create policy1.rego: %v", err)
	}

	file2 := filepath.Join(regoDir, "policy2.rego")
	err = os.WriteFile(file2, []byte(`package policy2

	default allow = true`), 0644)
	if err != nil {
		t.Fatalf("Failed to create policy2.rego: %v", err)
	}

	// Call AddRegoFilesFromDirectory
	newBundle, err := bundle.AddRegoFilesFromDirectory(&originalBundle, tempDir)
	if err != nil {
		t.Fatalf("Failed to add Rego files from directory: %v", err)
	}

	// Overwrite a file and add again
	err = os.WriteFile(file1, []byte(`package policy1
	default allow = true`), 0644)
	if err != nil {
		t.Fatalf("Failed to overwrite policy1.rego: %v", err)
	}
	newBundle, err = bundle.AddRegoFilesFromDirectory(newBundle, tempDir)
	if err != nil {
		t.Fatalf("Failed to add Rego files from directory: %v", err)
	}

	// Verify the new bundle contains the added modules
	if len(newBundle.Modules) != 2 {
		t.Fatalf("Expected 2 modules in the bundle, got %d", len(newBundle.Modules))
	}

	// Verify the paths of the added modules
	expectedPaths := []string{"serviceA/policy1.rego", "serviceA/policy2.rego"}
	for _, module := range bundle.ListBundleFiles(newBundle) {
		if !slices.Contains(expectedPaths, module) {
			t.Errorf("Unexpected module path: %s", module)
		}
	}

	// Verify the content of the added modules
	if string(newBundle.Modules[0].Raw) != `package policy1

	default allow = true` && string(newBundle.Modules[1].Raw) != `package policy2

	default allow = true` {
		t.Errorf("Module content does not match expected content")
	}
}
func TestRemoveService(t *testing.T) {
	// Create a mock bundle
	originalBundle, err := createBundleFromFiles(map[string]string{
		"serviceA/policy1.rego": `package serviceA.policy1
default allow = false`,
		"serviceA/policy2.rego": `package serviceA.policy2
default allow = true`,
		"serviceB/policy2.rego": `package serviceB.policy2
default allow = true`,
	}, "rego")
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	// Call RemoveService to remove serviceA
	newBundle, err := bundle.RemoveService(originalBundle, "serviceA")
	if err != nil {
		t.Fatalf("Failed to remove service: %v", err)
	}

	// Verify the new bundle does not contain modules from serviceA
	if len(newBundle.Modules) != 1 {
		t.Fatalf("Expected 1 module in the bundle, got %d", len(newBundle.Modules))
	}
	if newBundle.Modules[0].Path != "rego/serviceB/policy2.rego" {
		t.Fatalf("Expected remaining module to be 'rego/serviceB/policy2.rego', got '%s'", newBundle.Modules[0].Path)
	}

	// Verify the roots are updated
	if len(*newBundle.Manifest.Roots) != 1 {
		t.Fatalf("Expected 1 root in the bundle, got %d", len(*newBundle.Manifest.Roots))
	}
}

func TestRemoveServiceNonExistentSubdir(t *testing.T) {
	// Create a mock bundle
	originalBundle, err := createBundleFromFiles(map[string]string{
		"serviceA/policy1.rego": `package serviceA.policy1
default allow = false`,
		"serviceB/policy1.rego": `package serviceB.policy1
default allow = true`,
	}, "rego")
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	// Call RemoveService with a non-existent subdir
	newBundle, err := bundle.RemoveService(originalBundle, "serviceC")
	if err != nil {
		t.Fatalf("Failed to remove service: %v", err)
	}

	// Verify the new bundle is unchanged
	if len(newBundle.Modules) != 2 {
		t.Fatalf("Expected 2 modules in the bundle, got %d", len(newBundle.Modules))
	}
}

func TestRemoveServiceInvalidMainDir(t *testing.T) {
	// Create a mock bundle with no "main" metadata
	originalBundle, err := createBundleFromFiles(map[string]string{
		"serviceA/policy1.rego": `package serviceA.policy1
default allow = false`,
		"serviceB/policy2.rego": `package serviceB.policy2
default allow = true`,
	}, "rego")
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}
	// Remove the "main" metadata
	originalBundle.Manifest.Metadata = map[string]interface{}{}

	// Call RemoveService
	_, err = bundle.RemoveService(originalBundle, "serviceA")
	if err == nil {
		t.Fatal("Expected an error due to missing 'main' metadata, but got none")
	}
	if !strings.Contains(err.Error(), "failed to find main directory in bundle manifest") {
		t.Fatalf("Unexpected error message: %v", err)
	}
}

func createBundleFromFiles(regoFiles map[string]string, mainDir string) (*opabundle.Bundle, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "rego-bundle")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write the rego files to the temporary directory
	for fileName, content := range regoFiles {
		filePath := filepath.Join(tempDir, mainDir, fileName)
		dirPath := filepath.Dir(filePath)
		err := os.MkdirAll(dirPath, 0755) // Ensure the directory structure exists
		if err != nil {
			return nil, fmt.Errorf("failed to create directories for %s: %w", filePath, err)
		}
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write rego file %s: %w", fileName, err)
		}
	}

	// Create a bundle using opa compile
	comp := compile.New().
		WithAsBundle(true).
		WithFS(os.DirFS(tempDir)).
		WithMetadata(&map[string]interface{}{
			"main": mainDir,
		}).
		WithPaths(mainDir)
	err = comp.Build(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to create bundle: %w", err)
	}

	return comp.Bundle(), nil
}
