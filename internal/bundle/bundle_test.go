package bundle

import (
	policy "dspn-regogenerator/internal/policy"
	"os"
	"path/filepath"
	"slices"
	"testing"

	opabundle "github.com/open-policy-agent/opa/v1/bundle"
)

func prepareOpaBundle(t testing.TB, serviceNames []string, files map[string]string) opabundle.Bundle {
	t.Helper()

	// Store files in a temporary directory
	temp := t.TempDir()
	for path, content := range files {
		fullPath := temp + path
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0666); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}
	// Compile a bundle using standard OPA bundle reader
	loader, err := opabundle.NewFSLoader(os.DirFS(temp))
	bundleReader := opabundle.NewCustomReader(loader)
	bundle, err := bundleReader.Read()
	if err != nil {
		t.Fatalf("opaBundleReader.Read() error = %v", err)
	}
	if len(bundle.Modules) != len(files) {
		t.Fatalf("opaBundleReader.Read() returned %d modules, expected %d", len(bundle.Modules), len(files))
	}
	bundle.Manifest.Init()
	bundle.Manifest.Metadata = make(map[string]any)
	bundle.Manifest.Metadata["services"] = serviceNames
	bundle.Manifest.Roots = &serviceNames
	return bundle
}

func TestBundleFromService(t *testing.T) {
	service := Service{
		name:    "test_service",
		oidcUrl: "https://example.com/oidc",
		policy: policy.GeneralPolicies{
			Policies: []policy.PolicyClause{
				{
					UserPolicy: &policy.UserPolicy{
						Operator: policy.OperatorAnd,
						EnumeratedValue: policy.EnumeratedValue{
							Value: []string{"user1", "user2"},
						},
					},
				},
			},
		},
	}

	bundle, err := NewFromService(service)
	if err != nil {
		t.Fatalf("NewBundleFromService() error = %v", err)
	}

	if len(bundle.serviceNames) != 1 || bundle.serviceNames[0] != service.name {
		t.Errorf("NewBundleFromService() did not set service names correctly")
	}

	if len(bundle.bundle.Modules) != 2 {
		t.Errorf("NewBundleFromService() did not generate the expected number of modules, got %d, want 2", len(bundle.bundle.Modules))
	} else {
		urls := []string{bundle.bundle.Modules[0].URL, bundle.bundle.Modules[1].URL}
		if !slices.Contains(urls, "/test_service/service.rego") || !slices.Contains(urls, "/test_service/oidc.rego") {
			t.Errorf("NewBundleFromService() did not generate expected module URLs, got %v", urls)
		}
	}
	if _, ok := bundle.bundle.Manifest.Metadata["services"]; !ok {
		t.Error("NewBundleFromService() did not set services in manifest metadata")
	} else {
		services, ok := bundle.bundle.Manifest.Metadata["services"].([]string)
		if !ok || len(services) != 1 || services[0] != service.name {
			t.Errorf("NewBundleFromService() did not set correct service name in manifest metadata, got %v", services)
		}
	}

	if t.Failed() {
		t.FailNow()
	}

	files, err := service.generateServiceFiles()
	if err != nil {
		t.Fatalf("GenerateServiceFiles() error = %v", err)
	}
	standardBundle := prepareOpaBundle(t, bundle.serviceNames, files)

	// compare the modules of generated bundle and OPA bundle
	if len(standardBundle.Modules) != len(bundle.bundle.Modules) {
		t.Errorf("opaBundleReader.Read() returned %d modules, expected %d", len(standardBundle.Modules), len(bundle.bundle.Modules))
	} else {
		actualUrls := []string{bundle.bundle.Modules[0].URL, bundle.bundle.Modules[1].URL}
		t.Logf("custom build bundle has modules %v", actualUrls)
		for _, module := range standardBundle.Modules {
			t.Logf("opaBundleReader.Read() returned module %s", module.URL)
			if index := slices.Index(actualUrls, module.URL); index == -1 {
				t.Errorf("opaBundleReader.Read() returned module %s, expected %s", module.URL, actualUrls)
			} else if string(module.Raw) != string(bundle.bundle.Modules[index].Raw) {
				t.Errorf("opaBundleReader.Read() returned module %s content does not match generated bundle content", module.URL)
			}
		}
	}
}

func TestBundleFromTarball(t *testing.T) {
	generatedBundle := prepareOpaBundle(t, []string{"service1", "service2"}, map[string]string{
		"/service1/policy.rego": "package service1\n\n" +
			"request_policy if {\n" +
			"    \"valid\": valid,\n" +
			"}\n",
		"/service2/policy.rego": "package service2\n\n" +
			"request_policy if {\n" +
			"    \"valid\": valid,\n" +
			"}\n",
	})

	temp := t.TempDir()
	tarballPath := filepath.Join(temp, "bundle.tar.gz")
	file, err := os.Create(tarballPath)
	if err != nil {
		t.Fatalf("Failed to create tarball file %s: %v", tarballPath, err)
	}
	t.Cleanup(func() {
		file.Close()
	})
	writer := opabundle.NewWriter(file)
	if err := writer.Write(generatedBundle); err != nil {
		t.Fatalf("Failed to write bundle to tarball: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close tarball file %s: %v", tarballPath, err)
	}
	file, err = os.Open(tarballPath)
	if err != nil {
		t.Fatalf("Failed to open tarball file %s: %v", tarballPath, err)
	}

	loadedBundle, err := NewFromTarball(file)
	if err != nil {
		t.Fatalf("NewBundleFromTarball() error = %v", err)
	}
	t.Logf("Loaded bundle metadata: %v", loadedBundle.bundle.Manifest.Metadata)

	if len(loadedBundle.serviceNames) != 2 || !slices.Contains(loadedBundle.serviceNames, "service1") || !slices.Contains(loadedBundle.serviceNames, "service2") {
		t.Errorf("NewBundleFromTarball() did not set service names correctly, got %v", loadedBundle.serviceNames)
	}
	if len(loadedBundle.bundle.Modules) != 2 {
		t.Errorf("NewBundleFromTarball() did not generate the expected number of modules, got %d, want 2", len(loadedBundle.bundle.Modules))
	} else {
		urls := []string{loadedBundle.bundle.Modules[0].URL, loadedBundle.bundle.Modules[1].URL}
		if !slices.Contains(urls, "/service1/policy.rego") || !slices.Contains(urls, "/service2/policy.rego") {
			t.Errorf("NewBundleFromTarball() did not generate expected module URLs, got %v", urls)
		}
	}
}

func TestAddServiceToBundle(t *testing.T) {
	service := Service{
		name:    "new_service",
		oidcUrl: "https://example.com/oidc",
		policy: policy.GeneralPolicies{
			Policies: []policy.PolicyClause{
				{
					UserPolicy: &policy.UserPolicy{
						Operator: policy.OperatorAnd,
						EnumeratedValue: policy.EnumeratedValue{
							Value: []string{"user1", "user2"},
						},
					},
				},
			},
		},
	}

	opaBundle := prepareOpaBundle(t, []string{"test_service"}, map[string]string{
		"/test_service/service.rego": "package test_service\n\n" +
			"request_policy if {\n" +
			"    \"valid\": valid,\n" +
			"}\n",
		"/test_service/oidc.rego": "package test_service.oidc\n\n" +
			"metadata_url := \"https://example.com/oidc\"\n\n" +
			"metadata if {\n" +
			"    metadata_url := input.metadata_url\n" +
			"}\n",
	})

	bundle := &Bundle{
		bundle:       &opaBundle,
		serviceNames: []string{"test_service"},
	}

	err := bundle.AddService(service)
	if err != nil {
		t.Fatalf("AddService() error = %v", err)
	}

	if len(bundle.serviceNames) != 2 || !slices.Contains(bundle.serviceNames, service.name) {
		t.Errorf("AddService() did not update service names correctly, got %v", bundle.serviceNames)
	}
	if len(bundle.bundle.Modules) != 4 {
		t.Errorf("AddService() did not generate the expected number of modules, got %d, want 4", len(bundle.bundle.Modules))
	} else {
		urls := []string{bundle.bundle.Modules[0].URL, bundle.bundle.Modules[1].URL, bundle.bundle.Modules[2].URL, bundle.bundle.Modules[3].URL}
		if !slices.Contains(urls, "/new_service/service.rego") || !slices.Contains(urls, "/new_service/oidc.rego") {
			t.Errorf("AddService() did not generate expected module URLs, got %v", urls)
		}
	}
}

func TestRemoveServiceFromBundle(t *testing.T) {
	service := Service{
		name:    "test_service",
		oidcUrl: "https://example.com/oidc",
		policy: policy.GeneralPolicies{
			Policies: []policy.PolicyClause{
				{
					UserPolicy: &policy.UserPolicy{
						Operator: policy.OperatorAnd,
						EnumeratedValue: policy.EnumeratedValue{
							Value: []string{"user1", "user2"},
						},
					},
				},
			},
		},
	}

	opaBundle := prepareOpaBundle(t, []string{"test_service", "other_service"}, map[string]string{
		"/test_service/service.rego": "package test_service\n\n" +
			"request_policy if {\n" +
			"    \"valid\": valid,\n" +
			"}\n",
		"/test_service/oidc.rego": "package test_service.oidc\n\n" +
			"metadata_url := \"https://example.com/oidc\"\n\n" +
			"metadata if {\n" +
			"    metadata_url := input.metadata_url\n" +
			"}\n",
		"/other_service/service.rego": "package other_service\n\n" +
			"request_policy if {\n" +
			"    \"valid\": valid,\n" +
			"}\n",
		"/other_service/oidc.rego": "package other_service.oidc\n" +
			"metadata_url := \"https://example.com/oidc\"\n\n" +
			"metadata if {\n" +
			"    metadata_url := input.metadata_url\n" +
			"}\n",
	})

	bundle := &Bundle{
		bundle:       &opaBundle,
		serviceNames: []string{"test_service", "other_service"},
	}

	err := bundle.RemoveService(service.name)
	if err != nil {
		t.Fatalf("RemoveService() error = %v", err)
	}

	if len(bundle.serviceNames) != 1 {
		t.Errorf("RemoveService() did not remove service names correctly, got %v", bundle.serviceNames)
	}
	if len(bundle.bundle.Modules) != 2 {
		t.Errorf("RemoveService() did not remove modules correctly, got %d, want 2", len(bundle.bundle.Modules))
	}
}
