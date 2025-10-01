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

package bundle

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	opabundle "github.com/open-policy-agent/opa/v1/bundle"
)

func TestGetBundle(t *testing.T) {
	// Prepare file
	opaBundle := prepareOpaBundle(t, []string{"service1", "service2"}, map[string]string{
		"/service1/test.rego": "package service1\n\nallow := false\n",
		"/service2/test.rego": "package service2\n\nallow := true\n",
	})
	temp := t.TempDir()
	bundlePath := filepath.Join(temp, "bundle.tar.gz")
	file, err := os.Create(bundlePath)
	if err != nil {
		t.Fatalf("Error creating bundle file. %v", err)
	}
	err = opabundle.NewWriter(file).Write(*opaBundle)
	if err != nil {
		t.Fatalf("Error writing bundle file: %v", err)
	}

	// Test code
	repo := FSRepository{}
	bundle, err := repo.Get(bundlePath)

	// Assert result
	if err != nil {
		t.Errorf("Error getting the bundle: %v", err)
	}
	if bundle == nil {
		t.Fatal("Expected non null bundle")
	}
	if len(bundle.serviceNames) != 2 {
		t.Errorf("Expected bundle to have 2 elements, got %d", len(bundle.serviceNames))
	} else if !slices.Contains(bundle.serviceNames, "service1") || !slices.Contains(bundle.serviceNames, "service2") {
		t.Errorf("Expected bundle to have [\"service1\",\"service2\"], got %v", bundle.serviceNames)
	}
}

func TestSaveOpaBundle(t *testing.T) {
	// Prepare file
	opaBundle := prepareOpaBundle(t, []string{"service1", "service2"}, map[string]string{
		"/service1/test.rego": "package service1\n\nallow := false\n",
		"/service2/test.rego": "package service2\n\nallow := true\n",
	})
	temp := t.TempDir()
	bundlePath := filepath.Join(temp, "bundle.tar.gz")
	bundle := Bundle{
		bundle:       opaBundle,
		serviceNames: []string{"service1", "service2"},
	}

	// Test code
	repo := FSRepository{}
	err := repo.Save(bundlePath, &bundle)
	if err != nil {
		t.Errorf("Error saving the bundle: %v", err)
	}

	// Assert result
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		t.Errorf("Expected bundle file to exist at %s, but it does not", bundlePath)
	}
}
