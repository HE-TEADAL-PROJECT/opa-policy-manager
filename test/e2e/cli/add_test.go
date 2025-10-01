// Copyright 2025 Matteo Brambilla - TEADAL
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"slices"
	"testing"

	"github.com/open-policy-agent/opa/v1/bundle"
)

func bundleReader(t *testing.T, bundlePath string) *tar.Reader {
	t.Helper()
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

	return tar.NewReader(gzipReader)
}

func TestCreateNewBundle(t *testing.T) {
	bundlePath := t.TempDir() + "/test_bundle.tar.gz"
	serviceName := "test_service"
	cmd := exec.Command(binaryPath, "service", "add", serviceName, getProjectRoot()+"/testdata/schemas/httpbin-api.json", bundlePath, "--new")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute command: %v\nOutput: %s", err, string(output))
	}

	tarReader := bundleReader(t, bundlePath)

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

	tarReader := bundleReader(t, bundlePath)

	for file, err := tarReader.Next(); err == nil; file, err = tarReader.Next() {
		switch file.Name {
		case "/.manifest":
			data, err := io.ReadAll(tarReader)
			if err != nil {
				t.Fatalf("Failed to read .manifest content: %v", err)
			}
			manifest := bundle.Manifest{}
			err = json.Unmarshal(data, &manifest)

			if manifest.Roots == nil || len(*manifest.Roots) != 3 || !slices.Contains(*manifest.Roots, "test_service") || !slices.Contains(*manifest.Roots, "another_service") || !slices.Contains(*manifest.Roots, "main") {
				t.Fatalf("Manifest roots do not contain expected services: %v", manifest.Roots)
			}
			services, ok := manifest.Metadata["services"].([]any)
			if !ok {
				t.Fatalf("Manifest metadata 'services' is not a slice: %v", manifest.Metadata["services"])
			}
			if len(services) != 2 || !slices.Contains(services, "test_service") || !slices.Contains(services, "another_service") {
				t.Fatalf("Manifest metadata 'services' does not contain expected services: %v", manifest.Metadata["services"])
			}
		}
	}
}

func TestRemoveService(t *testing.T) {
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

	cmd = exec.Command(binaryPath, "service", "remove", "test_service", bundlePath)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to remove service from existing bundle: %v\nOutput: %s", err, string(output))
	}

	tarReader := bundleReader(t, bundlePath)

	for file, err := tarReader.Next(); err == nil; file, err = tarReader.Next() {
		switch file.Name {
		case "/.manifest":
			data, err := io.ReadAll(tarReader)
			if err != nil {
				t.Fatalf("Failed to read .manifest content: %v", err)
			}
			manifest := bundle.Manifest{}
			err = json.Unmarshal(data, &manifest)

			if manifest.Roots == nil || len(*manifest.Roots) != 2 || slices.Contains(*manifest.Roots, "test_service") || !slices.Contains(*manifest.Roots, "another_service") || !slices.Contains(*manifest.Roots, "main") {
				t.Fatalf("Manifest roots do not contain expected services: %v", manifest.Roots)
			}
			services, ok := manifest.Metadata["services"].([]any)
			if !ok {
				t.Fatalf("Manifest metadata 'services' is not a slice: %v", manifest.Metadata["services"])
			}
			if len(services) != 1 || slices.Contains(services, "test_service") || !slices.Contains(services, "another_service") {
				t.Fatalf("Manifest metadata 'services' does not contain expected services: %v", manifest.Metadata["services"])
			}
		}
	}
}
