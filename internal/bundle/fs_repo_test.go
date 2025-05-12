package bundle

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	opabundle "github.com/open-policy-agent/opa/v1/bundle"
)

func TestFileSystemRepository(t *testing.T) {
	t.Run("WriteBundle", func(t *testing.T) {
		tempDir := t.TempDir()
		bundleFileName := "test-bundle.tar.gz"
		repo := NewFileSystemRepository(tempDir)

		// Create a dummy bundle
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)
		bundle, err := NewFromFS(context.TODO(), os.DirFS(tempDir), "service1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Write the bundle to the repository
		err = repo.Write(bundleFileName, *bundle)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify the file exists
		fullPath := filepath.Join(tempDir, bundleFileName)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Fatalf("expected file %v to exist, but it does not", fullPath)
		}
		bundleFile, err := os.Open(fullPath)
		if err != nil {
			t.Fatalf("expected no error opening file, got %v", err)
		}
		defer bundleFile.Close()
		loadedBundle, err := NewFromArchive(context.TODO(), bundleFile)
		if err != nil {
			t.Fatalf("expected no error loading bundle, got %v", err)
		}
		if len(loadedBundle.bundle.Modules) != 1 {
			t.Fatalf("expected 1 module, got %d", len(loadedBundle.bundle.Modules))
		}
		if string(loadedBundle.bundle.Modules[0].Path) != "/service1/policy.rego" {
			t.Fatalf("expected module path to be 'service1/policy.rego', got '%s'", loadedBundle.bundle.Modules[0].Path)
		}
	})

	t.Run("WriteToInvalidPath", func(t *testing.T) {
		repo := NewFileSystemRepository("/invalid-path")
		mockBundle := &Bundle{
			bundle: &opabundle.Bundle{},
		}

		err := repo.Write("test-bundle.tar.gz", *mockBundle)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("ReadBundle", func(t *testing.T) {
		tempDir := t.TempDir()
		bundleFileName := "test-bundle.tar.gz"
		repo := NewFileSystemRepository(tempDir)

		// Create a dummy bundle
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)
		bundle, err := NewFromFS(context.TODO(), os.DirFS(tempDir), "service1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Create the bundle file
		bundleFile, err := os.Create(filepath.Join(tempDir, bundleFileName))
		if err != nil {
			t.Fatalf("expected no error creating file, got %v", err)
		}
		defer bundleFile.Close()
		writer := opabundle.NewWriter(bundleFile)
		if err := writer.Write(*bundle.bundle); err != nil {
			t.Fatalf("expected no error writing bundle, got %v", err)
		}
		bundleFile.Close()

		// Read the bundle from the repository
		loadedBundle, err := repo.Read(bundleFileName)
		if err != nil {
			t.Fatalf("expected no error reading bundle, got %v", err)
		}
		if len(loadedBundle.bundle.Modules) != 1 {
			t.Fatalf("expected 1 module, got %d", len(loadedBundle.bundle.Modules))
		}
		if string(loadedBundle.bundle.Modules[0].Path) != "/service1/policy.rego" {
			t.Fatalf("expected module path to be 'service1/policy.rego', got '%s'", loadedBundle.bundle.Modules[0].Path)
		}
	})

	t.Run("ReadNonExistentFile", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := NewFileSystemRepository(tempDir)

		_, err := repo.Read("non-existent-bundle.tar.gz")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
