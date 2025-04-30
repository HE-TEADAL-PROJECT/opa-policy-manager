package config

import (
	"fmt"
	"os"
	"strconv"
)

func GetEnvOrDefault(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

var (
	// Address of the MinIO server, without the protocol (http:// or https://).
	// The default value is "localhost:9000", load from environment variable MINIO_SERVER.
	MinioEndpoint string

	// The access key for MinIO.
	// The default value is "admin", load from environment variable MINIO_ACCESS_KEY.
	MinioAccessKey string

	// The secret key for MinIO.
	// The default value is "adminadmin", load from environment variable MINIO_SECRET_KEY.
	MinioSecretKey string

	// The bucket name where to store the bundles.
	// The default value is "opa-policy-bundles", load from environment variable BUCKET_NAME.
	MinioBucket string

	// The bundle name prefix, used to create the bundle name adding a -version tag suffix.
	// The default value is "teadal-policy-bundle", load from environment variable MINIO_BUNDLE_PREFIX.
	MinioBundlePrefix string

	// The name of the latest bundle.
	LatestBundleName string

	// A function to generate a bundle name with a specific tag.
	TagBundleName func(tag string) string

	// The timeout for MinIO operations in seconds.
	// The default value is 5 seconds, load from environment variable MINIO_TIMEOUT.
	MinioTimeout int
)

// ReloadConfig initializes or reloads the global variables based on the current environment variables. There is no need to call this function manually, as it is automatically called when the package is loaded.
func ReloadConfig() {
	MinioEndpoint = GetEnvOrDefault("MINIO_SERVER", "localhost:9000")
	MinioAccessKey = GetEnvOrDefault("MINIO_ACCESS_KEY", "admin")
	MinioSecretKey = GetEnvOrDefault("MINIO_SECRET_KEY", "adminadmin")
	MinioBucket = GetEnvOrDefault("BUCKET_NAME", "opa-policy-bundles")
	MinioBundlePrefix = GetEnvOrDefault("MINIO_BUNDLE_PREFIX", "teadal-policy-bundle")
	LatestBundleName = MinioBundlePrefix + "-LATEST.tar.gz"
	TagBundleName = func(tag string) string {
		return MinioBundlePrefix + "-" + tag + ".tar.gz"
	}
	var err error
	MinioTimeout, err = strconv.Atoi(GetEnvOrDefault("MINIO_TIMEOUT", "5"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing MINIO_TIMEOUT: %v\n", err)
		MinioTimeout = 5
	}
}

func init() {
	ReloadConfig()
}
