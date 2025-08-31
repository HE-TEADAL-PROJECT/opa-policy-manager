package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"os/exec"
	"slices"
	"testing"

	"github.com/open-policy-agent/opa/v1/bundle"
)

func TestCreateNewBundle(t *testing.T) {
	bundlePath := t.TempDir() + "/test_bundle.tar.gz"
	serviceName := "test_service"
	cmd := exec.Command(binaryPath, "service", "add", serviceName, getProjectRoot()+"/testdata/schemas/httpbin-api.json", bundlePath, "--new")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute command: %v\nOutput: %s", err, string(output))
	}

	file, err := os.Open(bundlePath)
	if err != nil {
		t.Fatalf("Failed to open bundle file: %v", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for file, err := tarReader.Next(); err == nil; file, err = tarReader.Next() {
		switch file.Name {
		case "/data.json":
		case "/" + serviceName + "/oidc.rego":
		case "/" + serviceName + "/service.rego":
		case "/main.rego":
		case "/.manifest":
			buffer := bytes.Buffer{}
			_, err := buffer.ReadFrom(tarReader)
			if err != nil {
				t.Fatalf("Failed to read .manifest content: %v", err)
			}
			content := buffer.Bytes()
			manifest := bundle.Manifest{}
			err = json.Unmarshal(content, &manifest)
			if err != nil {
				t.Fatalf("Failed to unmarshal .manifest content: %v", err)
			}
			if !slices.Contains(*manifest.Roots, serviceName) || !slices.Contains(*manifest.Roots, "main") {
				t.Fatalf("Manifest roots do not contain expected services: %v", manifest.Roots)
			}
			services, ok := manifest.Metadata["services"].([]any)
			if !ok {
				t.Fatalf("Manifest metadata 'services' is not a slice: %v", manifest.Metadata["services"])
			}
			if len(services) != 1 || services[0] != serviceName {
				t.Fatalf("Manifest metadata 'services' does not contain expected service: %v", manifest.Metadata["services"])
			}
			t.Log("Manifest file as expected")
		default:
			t.Fatalf("Unexpected file in bundle: %s", file.Name)
		}
	}
}

func TestAddService(t *testing.T) {
	bundlePath := t.TempDir() + "/test_bundle.tar.gz"
	cmd := exec.Command(binaryPath, "service", "add", "test_service", getProjectRoot()+"/testdata/schemas/httpbin-api.json", bundlePath, "--new")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create new bundle: %v\nOutput: %s", err, string(output))
	}

	cmd = exec.Command(binaryPath, "service", "add", "another_service", getProjectRoot()+"/testdata/schemas/httpbin-api.json", bundlePath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to add service to existing bundle: %v\nOutput: %s", err, string(output))
	}
}
