package usecases

import (
	"context"
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"dspn-regogenerator/internal/generator"
	"dspn-regogenerator/internal/policy/parser"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/storage/inmem"
	"github.com/open-policy-agent/opa/tester"
)

var testFile string = `package testBundle_test

import data.testBundle.oidc
import data.testBundle

test_expected_metadata if {
    oidc.metadata_url != null
}

test_allow_get_bearer if {
    testBundle.allow with oidc.token as {
        "valid": true,
        "payload": {
            "preferred_username": "jeejee@teadal.eu",
            "realm_access": {
                "roles": ["role1", "role2"]
            }
        }
    } with data.input.attributes.request.http as {
        "path": "/bearer",
        "method": "get",
    }
}`

var anonymousPolicy string = `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": [
                    "*"
                ]
            },
            "Action": [
                "s3:GetBucketLocation",
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::%s"
            ]
        },
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": [
                    "*"
                ]
            },
            "Action": [
                "s3:GetObject"
            ],
            "Resource": [
                "arn:aws:s3:::%s/*"
            ]
        }
    ]
}`

func testMinioConnection(ctx context.Context) error {
	client, err := minio.New(config.MinioEndpoint, &minio.Options{
		Creds: credentials.NewStaticV4(config.MinioAccessKey, config.MinioSecretKey, "")})
	if err != nil {
		return err
	}

	_, err = client.HealthCheck(time.Duration(config.MinioTimeout) * time.Second)

	if bucketExists, err := client.BucketExists(ctx, config.MinioBucket); err != nil {
		return err
	} else if !bucketExists {
		err := client.MakeBucket(ctx, config.MinioBucket, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
		err = client.SetBucketPolicy(ctx, config.MinioBucket, fmt.Sprintf(anonymousPolicy, config.MinioBucket, config.MinioBucket))
		if err != nil {
			return err
		}
	}

	return err
}

func loadSpecFile(specFile string) ([]byte, error) {
	// Load the OpenAPI spec file
	specData, err := os.ReadFile(specFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI spec file: %v", err)
	}
	return specData, nil
}

func InitialTest(ctx context.Context) error {
	// Test parameters
	serviceName := "testBundle"
	testSchemaPath := "./testdata/schemas/httpbin-api.json"

	// Test the MinIO connection
	if err := testMinioConnection(ctx); err != nil {
		return fmt.Errorf("error connecting to MinIO: %w", err)
	}
	slog.Info("Connected to MinIO successfully")

	// Verify if the bundle exists on minio
	bundleExists, err := bundle.CheckBundleFileExists(config.LatestBundleName)
	if err != nil {
		return fmt.Errorf("error checking if service exists: %w", err)
	}
	if !bundleExists {

		// Load the OpenAPI spec file
		specData, err := loadSpecFile(testSchemaPath)
		if err != nil {
			return fmt.Errorf("error loading OpenAPI spec file: %w", err)
		}

		// Parse the OpenAPI spec to extract policies and provider
		policies, err := parser.ParseOpenAPIPolicies(specData)
		if err != nil || policies == nil {
			return fmt.Errorf("error parsing OpenAPI spec: %w", err)
		}
		provider, err := parser.ParseOpenAPIIAM(specData)
		if err != nil || provider == nil {
			return fmt.Errorf("error parsing OpenAPI provider: %w", err)
		}

		// Create a temporary directory for the output
		tempDir, err := os.MkdirTemp("", "bundle-*")
		if err != nil {
			return fmt.Errorf("error creating temp directory: %w", err)
		}
		regoDir := filepath.Join(tempDir, "rego")
		err = os.MkdirAll(regoDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating rego directory: %w", err)
		}

		// Generate the static folder
		generator.GenerateStaticFolders(regoDir)

		// Generate the service folder
		err = generator.GenerateServiceFolder(serviceName, regoDir, *provider, policies)
		if err != nil {
			return fmt.Errorf("error generating service folder: %w", err)
		}

		// Write the test file
		testFilePath := filepath.Join(regoDir, serviceName, "test_test.rego")
		err = os.WriteFile(testFilePath, []byte(testFile), 0644)
		if err != nil {
			return fmt.Errorf("error writing test file: %w", err)
		}

		// Build the bundle
		b, err := bundle.BuildBundle(tempDir, "rego")
		if err != nil {
			return fmt.Errorf("error building bundle: %w", err)
		}
		if err := bundle.WriteBundleToMinio(b, config.LatestBundleName); err != nil {
			return fmt.Errorf("error writing bundle to Minio: %w", err)
		}
		slog.Info("Bundle written to MinIO successfully")
	}

	// Download the bundle from MinIO
	bundlePath, err := bundle.LoadBundleFromMinio(config.LatestBundleName)
	if err != nil {
		return fmt.Errorf("error downloading bundle from MinIO: %w", err)
	}
	testBundlePath := filepath.Join(os.TempDir(), config.LatestBundleName)
	err = bundle.WriteBundleToFile(bundlePath, testBundlePath)
	if err != nil {
		return fmt.Errorf("error writing bundle to file: %w", err)
	}

	// Load the bundle and execute the tests
	bundles, err := tester.LoadBundles([]string{testBundlePath}, nil)
	if err != nil {
		return fmt.Errorf("error loading bundle: %w", err)
	}
	store := inmem.New()

	testRunner := tester.NewRunner()
	testRunner.SetBundles(bundles)
	testRunner.SetStore(store)
	ch, err := testRunner.RunTests(ctx, storage.NewTransactionOrDie(ctx, store, storage.WriteParams))
	if err != nil {
		return fmt.Errorf("error running tests: %w", err)
	}
	var failed = false
	for result := range ch {
		if result.Error != nil || result.Fail {
			failed = true
			slog.Error(fmt.Sprintf("%s failed", result.Name), "error", result.Error, "package", result.Package)
		}
	}
	if failed {
		return fmt.Errorf("some tests failed")
	}
	slog.Info("All rego tests passed successfully")

	return nil
}
