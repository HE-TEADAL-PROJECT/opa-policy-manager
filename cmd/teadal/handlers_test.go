package main

import (
	"bytes"
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/policy/parser"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
)

const globalTestdataDir = "../../testdata"

type mockBundleRepository struct {
	bundle *bundle.Bundle
}

func (m *mockBundleRepository) Get(path string) (*bundle.Bundle, error) {
	return m.bundle, nil
}

func (m *mockBundleRepository) Save(path string, b *bundle.Bundle) error {
	m.bundle = b
	return nil
}

func TestDescribeBundleHandler(t *testing.T) {
	b, err := bundle.New(bundle.NewService("test_service", &parser.ServiceSpec{}))
	if err != nil {
		t.Fatalf("failed to create bundle: %v", err)
	}

	repo := &mockBundleRepository{bundle: b}
	lock := sync.RWMutex{}
	handler := describeBundleHandler(repo, &lock)

	r := httptest.NewRequest(http.MethodGet, "http://localhost/api/policies", nil)
	w := httptest.NewRecorder()
	handler(w, r)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status OK, got %v", resp.Status)
	}

	serviceNames := []string{}
	json.NewDecoder(resp.Body).Decode(&serviceNames)
	if len(serviceNames) != 1 || serviceNames[0] != "test_service" {
		t.Fatalf("expected service names to contain 'test_service', got %v", serviceNames)
	}
}

func TestAddServiceHandler(t *testing.T) {
	b, err := bundle.New(bundle.NewService("test_service", &parser.ServiceSpec{}))
	if err != nil {
		t.Fatalf("failed to create bundle: %v", err)
	}
	bundleRepo := &mockBundleRepository{
		bundle: b,
	}
	lock := sync.RWMutex{}
	handler := addServiceHandler(bundleRepo, &lock)

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile(specFileFieldName, "httpbin-api.json")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	specFile, err := os.Open("../../testdata/schemas/httpbin-api.json")
	if err != nil {
		t.Fatalf("failed to open spec file: %v", err)
	}
	defer specFile.Close()
	if _, err = io.Copy(fw, specFile); err != nil {
		t.Fatalf("failed to copy spec file: %v", err)
	}
	if err = mw.WriteField(serviceNameFieldName, "httpbin"); err != nil {
		t.Fatalf("failed to write service name field: %v", err)
	}
	if err = mw.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	t.Log("Multipart writer completed")

	req := httptest.NewRequest(http.MethodPost, "http://localhost/api/policies", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Error: %v", string(body))
		t.Fatalf("expected status CREATED, got %v", resp.Status)
	}

	b = bundleRepo.bundle
	if !slices.Contains(b.Describe(), "httpbin") {
		t.Errorf("expected bundle to contain service 'httpbin', got %v", b.Describe())
	}
}

func TestDeleteServiceHandler(t *testing.T) {
	// create the bundle
	file, err := os.Open(filepath.Join(globalTestdataDir, "schemas", "httpbin-api.json"))
	if err != nil {
		t.Fatalf("failed to open spec file: %v", err)
	}

	parsedSpec, err := parser.ParseServiceSpec(file)
	if err != nil {
		t.Fatalf("failed to parse OpenAPI spec: %v", err)
	}
	file.Close()

	service := bundle.NewService("httpbin", parsedSpec)
	b, err := bundle.New(service)
	if err != nil {
		t.Fatalf("failed to create bundle: %v", err)
	}

	bundleRepo := &mockBundleRepository{
		bundle: b,
	}
	lock := sync.RWMutex{}
	handler := deleteServiceHandler(bundleRepo, &lock)

	req := httptest.NewRequest(http.MethodDelete, "http://localhost/api/policies?serviceName=httpbin", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status OK, got %v", resp.Status)
	}

	b = bundleRepo.bundle
	if slices.Contains(b.Describe(), "httpbin") {
		t.Errorf("expected bundle to not contain service 'httpbin', got %v", b.Describe())
	}
}
