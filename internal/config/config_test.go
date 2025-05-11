package config

import "testing"

func TestLoadDefaultConfig(t *testing.T) {
	if MinioEndpoint != "localhost:9000" {
		t.Errorf("Expected MinioEndpoint to be 'localhost:9000', got '%s'", MinioEndpoint)
	}
	if MinioAccessKey != "admin" {
		t.Errorf("Expected MinioAccessKey to be 'admin', got '%s'", MinioAccessKey)
	}
	if MinioSecretKey != "adminadmin" {
		t.Errorf("Expected MinioSecretKey to be 'adminadmin', got '%s'", MinioSecretKey)
	}
	if MinioBucket != "opa-policy-bundles" {
		t.Errorf("Expected MinioBucket to be 'opa-policy-bundles', got '%s'", MinioBucket)
	}
	if MinioBundlePrefix != "teadal-policy-bundle" {
		t.Errorf("Expected MinioBundlePrefix to be 'teadal-policy-bundle', got '%s'", MinioBundlePrefix)
	}
	if LatestBundleName != "teadal-policy-bundle-LATEST.tar.gz" {
		t.Errorf("Expected LatestBundleName to be 'teadal-policy-bundle-LATEST.tar.gz', got '%s'", LatestBundleName)
	}
	if TagBundleName("v1") != "teadal-policy-bundle-v1.tar.gz" {
		t.Errorf("Expected TagBundleName('v1') to be 'teadal-policy-bundle-v1.tar.gz', got '%s'", TagBundleName("v1"))
	}
}

func TestLoadEnvConfig(t *testing.T) {
	t.Setenv("MINIO_SERVER", "test-endpoint")
	t.Setenv("MINIO_ACCESS_KEY", "test-access-key")
	t.Setenv("MINIO_SECRET_KEY", "test-secret-key")
	t.Setenv("BUCKET_NAME", "test-bucket")
	t.Setenv("MINIO_BUNDLE_PREFIX", "test-bundle-prefix")
	ReloadConfig()
	if MinioEndpoint != "test-endpoint" {
		t.Errorf("Expected MinioEndpoint to be 'test-endpoint', got '%s'", MinioEndpoint)
	}
	if MinioAccessKey != "test-access-key" {
		t.Errorf("Expected MinioAccessKey to be 'test-access-key', got '%s'", MinioAccessKey)
	}
	if MinioSecretKey != "test-secret-key" {
		t.Errorf("Expected MinioSecretKey to be 'test-secret-key', got '%s'", MinioSecretKey)
	}
	if MinioBucket != "test-bucket" {
		t.Errorf("Expected MinioBucket to be 'test-bucket', got '%s'", MinioBucket)
	}
	if MinioBundlePrefix != "test-bundle-prefix" {
		t.Errorf("Expected MinioBundlePrefix to be 'test-bundle-prefix', got '%s'", MinioBundlePrefix)
	}
	if LatestBundleName != "test-bundle-prefix-LATEST.tar.gz" {
		t.Errorf("Expected LatestBundleName to be 'test-bundle-prefix-LATEST.tar.gz', got '%s'", LatestBundleName)
	}
	if TagBundleName("v1") != "test-bundle-prefix-v1.tar.gz" {
		t.Errorf("Expected TagBundleName('v1') to be 'test-bundle-prefix-v1.tar.gz', got '%s'", TagBundleName("v1"))
	}
}
