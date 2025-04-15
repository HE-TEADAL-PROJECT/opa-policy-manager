package bundle_test

import (
	"context"
	"dspn-regogenerator/internal/bundle"
	"os"
	"path/filepath"
	"testing"

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
