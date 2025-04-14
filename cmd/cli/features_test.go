package main

import (
	"os"
	"testing"
)

func TestLoadLocalBundle(t *testing.T) {
	os.Chdir("../..")
	bundleURI := "./testdata/test-bundle.tar.gz"
	bundleData, err := loadBundle(bundleURI)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(bundleData) == 0 {
		t.Fatalf("expected non-empty bundle data, got empty")
	}
}

func TestLoadRemoteBundle(t *testing.T) {
	bundleURI := "https://example.com/test-bundle.tar.gz"
	_, err := loadBundle(bundleURI)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
