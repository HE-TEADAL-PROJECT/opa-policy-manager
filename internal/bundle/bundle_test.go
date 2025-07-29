package bundle

import (
	"bytes"
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

	testedBundle, err := NewFromService(service)
	if err != nil {
		t.Fatalf("NewBundleFromService() error = %v", err)
	}

	if len(testedBundle.serviceNames) != 1 || testedBundle.serviceNames[0] != service.name {
		t.Errorf("NewBundleFromService() did not set service names correctly")
	}

	testedBundleUrls := make([]string, len(testedBundle.bundle.Modules))
	for i, module := range testedBundle.bundle.Modules {
		testedBundleUrls[i] = module.URL
	}
	expectedUrls := []string{
		"/test_service/service.rego",
		"/test_service/oidc.rego",
		"/main.rego",
	}
	if len(testedBundleUrls) != len(expectedUrls) {
		t.Errorf("NewBundleFromService() did not generate expected module URLs, got %v, want %v", testedBundleUrls, expectedUrls)
	}
	for _, expectedUrl := range expectedUrls {
		if !slices.Contains(testedBundleUrls, expectedUrl) {
			t.Errorf("NewBundleFromService() did not generate expected module URL %s, got %v", expectedUrl, testedBundleUrls)
		}
	}

	if _, ok := testedBundle.bundle.Manifest.Metadata["services"]; !ok {
		t.Error("NewBundleFromService() did not set services in manifest metadata")
	} else {
		services, ok := testedBundle.bundle.Manifest.Metadata["services"].([]string)
		if !ok || len(services) != 1 || services[0] != service.name {
			t.Errorf("NewBundleFromService() did not set correct service name in manifest metadata, got %v", services)
		}
	}

	if t.Failed() {
		t.FailNow()
	}

	files, err := service.generateServiceFiles()
	files["/main.rego"] = generateMainFile(testedBundle.serviceNames)
	if err != nil {
		t.Fatalf("GenerateServiceFiles() error = %v", err)
	}
	standardBundle := prepareOpaBundle(t, testedBundle.serviceNames, files)

	for _, module := range standardBundle.Modules {
		index := slices.Index(testedBundleUrls, module.URL)
		if index == -1 {
			t.Errorf("Expected Module %s not found in generated bundle", module.URL)
		} else if !slices.Equal(module.Raw, testedBundle.bundle.Modules[index].Raw) {
			t.Errorf("Module %s content does not match", module.URL)
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
		"/main.rego": "package main\n\n" +
			"import data.test_service\n\n" +
			"default allow := false\n\n" +
			"allow if test_service.request_policy\n",
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
	if len(bundle.bundle.Modules) != 5 {
		t.Errorf("AddService() did not generate the expected number of modules, got %d, want 5", len(bundle.bundle.Modules))
	} else {
		urls := make([]string, len(bundle.bundle.Modules))
		for i, module := range bundle.bundle.Modules {
			urls[i] = module.URL
		}
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
		"/main.rego": "package main\n\n" +
			"import data.test_service\n" +
			"import data.other_service\n\n" +
			"default allow := false\n\n" +
			"allow if test_service.request_policy\n" +
			"allow if other_service.request_policy\n",
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
	if len(bundle.bundle.Modules) != 3 {
		t.Errorf("RemoveService() did not remove modules correctly, got %d, want 3", len(bundle.bundle.Modules))
	}

	index := slices.IndexFunc(bundle.bundle.Modules, func(m opabundle.ModuleFile) bool {
		return m.URL == mainFilePath
	})
	if index == -1 {
		t.Errorf("RemoveService() did not generate main.rego file")
	} else {
		mainModule := bundle.bundle.Modules[index]
		if bytes.Contains(mainModule.Raw, []byte("import data.test_service")) || bytes.Contains(mainModule.Raw, []byte("allow if test_service.request_policy")) {
			t.Errorf("RemoveService() did not remove other_service import from main.rego")
		}
	}

}
