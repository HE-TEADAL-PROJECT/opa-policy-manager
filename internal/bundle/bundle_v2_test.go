package bundle

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/open-policy-agent/opa/v1/bundle"
)

func TestNewFromFS(t *testing.T) {
	t.Run("SingleFileService", func(t *testing.T) {
		// Create a temporary file system with a single file service
		// and test the NewFromFS function.
		tempDir := t.TempDir()
		tempFS := os.DirFS(tempDir)
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)

		// Create a bundle from the temporary file system
		bundle, err := NewFromFS(context.Background(), tempFS, "service1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if bundle == nil {
			t.Fatal("expected bundle to be non-nil")
		}

		// Check if the services are correctly identified
		services, err := bundle.Services()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(services) != 1 || services[0] != "service1" {
			t.Fatalf("expected service1, got %v", services)
		}

		// Check if opa bundle is correct
		if bundle.bundle == nil {
			t.Fatal("expected bundle to be non-nil")
		}
		if len(bundle.bundle.Modules) != 1 {
			t.Fatalf("expected 1 module, got %v", len(bundle.bundle.Modules))
		}
		if bundle.bundle.Modules[0].Path != "/service1/policy.rego" {
			t.Fatalf("expected /service1/policy.rego, got %v", bundle.bundle.Modules[0].Path)
		}
	})

	t.Run("MultipleFileService", func(t *testing.T) {
		// Create a temporary file system with multiple file services
		// and test the NewFromFS function.
		tempDir := t.TempDir()
		tempFS := os.DirFS(tempDir)
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)
		os.Mkdir(tempDir+"/service2", 0755)
		os.WriteFile(tempDir+"/service2/policy.rego", []byte("package service2\n"), 0644)

		// Create a bundle from the temporary file system
		bundle, err := NewFromFS(context.Background(), tempFS, "service1", "service2")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if bundle == nil {
			t.Fatal("expected bundle to be non-nil")
		}

		// Check if the services are correctly identified
		services, err := bundle.Services()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(services) != 2 {
			t.Fatalf("expected 2 services, got %v", len(services))
		}

		if bundle.bundle == nil {
			t.Fatal("expected bundle to be non-nil")
		}
		if len(bundle.bundle.Modules) != 2 {
			t.Fatalf("expected 2 modules, got %v", len(bundle.bundle.Modules))
		}
		if bundle.bundle.Modules[0].Path != "/service1/policy.rego" {
			t.Fatalf("expected /service1/policy.rego, got %v", bundle.bundle.Modules[0].Path)
		}
		if bundle.bundle.Modules[1].Path != "/service2/policy.rego" {
			t.Fatalf("expected /service2/policy.rego, got %v", bundle.bundle.Modules[1].Path)
		}
	})

	t.Run("InvalidServiceFile", func(t *testing.T) {
		// Create a temporary file system with an invalid service file
		// and test the NewFromFS function.
		tempDir := t.TempDir()
		tempFS := os.DirFS(tempDir)
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)
		os.WriteFile(tempDir+"/service1/invalid.rego", []byte("invalid content\n"), 0644)

		// Create a bundle from the temporary file system
		bundle, err := NewFromFS(context.Background(), tempFS, "service1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if bundle != nil {
			t.Fatal("expected bundle to be nil")
		}
	})

	t.Run("MainFile", func(t *testing.T) {
		// Create a temporary file system with a main file
		// and test the NewFromFS function.
		tempDir := t.TempDir()
		tempFS := os.DirFS(tempDir)
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)
		os.WriteFile(tempDir+"/main.rego", []byte("package main\n"), 0644)

		bundle, err := NewFromFS(context.Background(), tempFS, "service1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(bundle.bundle.Modules) != 2 {
			t.Fatalf("expected 2 modules, got %v", len(bundle.bundle.Modules))
		}
		if bundle.bundle.Modules[0].Path != "/main.rego" && bundle.bundle.Modules[1].Path != "/main.rego" {
			t.Fatalf("expected /main.rego, got %v", bundle.bundle.Modules[0].Path)
		}
	})
}

func TestNewFromArchive(t *testing.T) {
	t.Run("ValidArchive", func(t *testing.T) {
		// Create a temporary archive with a valid OPA bundle
		// and test the NewFromArchive function.
		tempDir := t.TempDir()
		tempFS := os.DirFS(tempDir)
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)
		metadata := map[string]interface{}{
			"services": []string{"service1"},
		}

		b, err := bundle.NewCustomReader(bundle.NewFSLoaderWithRoot(tempFS, ".")).Read()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		b.Manifest.Metadata = metadata
		archive := &bytes.Buffer{}
		if err := bundle.NewWriter(archive).Write(b); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Create a bundle from the temporary archive
		bundle, err := NewFromArchive(context.Background(), archive)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if bundle == nil {
			t.Fatal("expected bundle to be non-nil")
		}

		// Check if the services are correctly identified
		services, err := bundle.Services()
		if err != nil {
			t.Logf("bundle metadata: %v", bundle.bundle.Manifest.Metadata)
			if bundle.bundle.Manifest.Metadata != nil {
				t.Logf("bundle metadata: %T", bundle.bundle.Manifest.Metadata["services"])
				services, ok := bundle.bundle.Manifest.Metadata["services"].([]string)
				t.Logf("casting services (ok %v): %v", ok, services)
			}
			t.Fatalf("expected no error, got %v", err)
		}
		if len(services) != 1 || services[0] != "service1" {
			t.Fatalf("expected service1, got %v", services)
		}

		if bundle.bundle == nil {
			t.Fatal("expected bundle to be non-nil")
		}
		if len(bundle.bundle.Modules) != 1 {
			t.Fatalf("expected 1 module, got %v", len(bundle.bundle.Modules))
		}
		if bundle.bundle.Modules[0].Path != "/service1/policy.rego" {
			t.Fatalf("expected /service1/policy.rego, got %v", bundle.bundle.Modules[0].Path)
		}
	})

	t.Run("WithMainFile", func(t *testing.T) {
		// Create a temporary archive with a main file
		// and test the NewFromArchive function.
		tempDir := t.TempDir()
		tempFS := os.DirFS(tempDir)
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)
		os.WriteFile(tempDir+"/main.rego", []byte("package main\n"), 0644)

		metadata := map[string]interface{}{
			"services": []string{"service1"},
		}

		// Create a bundle from the temporary archive
		b, err := bundle.NewCustomReader(bundle.NewFSLoaderWithRoot(tempFS, ".")).Read()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		b.Manifest.Metadata = metadata
		archive := &bytes.Buffer{}
		if err := bundle.NewWriter(archive).Write(b); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		bundle, err := NewFromArchive(context.Background(), archive)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(bundle.bundle.Modules) != 2 {
			t.Fatalf("expected 2 modules, got %v", len(bundle.bundle.Modules))
		}
		if bundle.bundle.Modules[0].Path != "/main.rego" && bundle.bundle.Modules[1].Path != "/main.rego" {
			t.Fatalf("expected /main.rego, got %v", bundle.bundle.Modules[0].Path)
		}
	})
}

func TestAddService(t *testing.T) {
	beforeEach := func(t *testing.T) (*Bundle, error) {
		tempDir := t.TempDir()
		tempFS := os.DirFS(tempDir)
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)

		bundle, err := NewFromFS(context.Background(), tempFS, "service1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		return bundle, nil
	}

	t.Run("AddNewService", func(t *testing.T) {
		bundle, err := beforeEach(t)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		specData := map[string][]byte{
			"service2/policy.rego": []byte("package service2\n"),
		}
		err = bundle.AddService("service2", specData)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check if the services are correctly identified
		services, err := bundle.Services()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(services) != 2 || services[0] != "service1" || services[1] != "service2" {
			t.Fatalf("expected service1 and service2, got %v", services)
		}

		// Check module loaded
		if len(bundle.bundle.Modules) != 2 {
			t.Fatalf("expected 2 modules, got %v", len(bundle.bundle.Modules))
		}
		if bundle.bundle.Modules[1].Path != "/service2/policy.rego" {
			t.Fatalf("expected /service2/policy.rego, got %v", bundle.bundle.Modules[1].Path)
		}
	})

	t.Run("UpdateExistingService", func(t *testing.T) {
		bundle, err := beforeEach(t)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		specData := map[string][]byte{
			"service1/policy.rego": []byte("package service1\n"),
		}
		err = bundle.AddService("service1", specData)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check if the services are correctly identified
		services, err := bundle.Services()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(services) != 1 || services[0] != "service1" {
			t.Fatalf("expected service1, got %v", services)
		}

		if len(bundle.bundle.Modules) != 1 {
			t.Fatalf("expected 1 module, got %v", len(bundle.bundle.Modules))
		}
		if bundle.bundle.Modules[0].Path != "/service1/policy.rego" {
			t.Fatalf("expected /service1/policy.rego, got %v", bundle.bundle.Modules[0].Path)
		}
	})
}

func TestGetMain(t *testing.T) {
	t.Run("ValidMain", func(t *testing.T) {
		// Create a temporary file system with a valid main file
		// and test the GetMain function.
		tempDir := t.TempDir()
		tempFS := os.DirFS(tempDir)
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)
		os.WriteFile(tempDir+"/main.rego", []byte("package main\n"), 0644)

		bundle, err := NewFromFS(context.Background(), tempFS, "service1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		main, err := bundle.GetMain()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if main == nil {
			t.Fatal("expected main to be non-nil")
		}
		if bytes.Equal(main, []byte("package main\n")) == false {
			t.Fatalf("expected main content to be 'package main\\n', got %v", string(main))
		}
	})

	t.Run("NoMainFile", func(t *testing.T) {
		// Create a temporary file system without a main file
		// and test the GetMain function.
		tempDir := t.TempDir()
		tempFS := os.DirFS(tempDir)
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)

		bundle, err := NewFromFS(context.Background(), tempFS, "service1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		main, err := bundle.GetMain()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if main != nil {
			t.Fatal("expected main to be nil")
		}
	})
}

func TestRemoveService(t *testing.T) {
	t.Run("RemoveExistingService", func(t *testing.T) {
		// Create a temporary file system with a service
		// and test the RemoveService function.
		tempDir := t.TempDir()
		tempFS := os.DirFS(tempDir)
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)

		bundle, err := NewFromFS(context.Background(), tempFS, "service1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		err = bundle.RemoveService("service1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check if the services are correctly identified
		services, err := bundle.Services()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(services) != 0 {
			t.Fatalf("expected no services, got %v", services)
		}

		// Check if the bundle is empty
		if bundle.bundle == nil {
			t.Fatal("expected bundle to be non-nil")
		}
		if len(bundle.bundle.Modules) != 0 {
			t.Fatalf("expected 0 modules, got %v", len(bundle.bundle.Modules))
		}
	})

	t.Run("RemoveNonExistingService", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFS := os.DirFS(tempDir)
		os.Mkdir(tempDir+"/service1", 0755)
		os.WriteFile(tempDir+"/service1/policy.rego", []byte("package service1\n"), 0644)

		bundle, err := NewFromFS(context.Background(), tempFS, "service1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		err = bundle.RemoveService("non-existing-service")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
