package usecases

import (
	"context"
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"dspn-regogenerator/internal/generator"
	"dspn-regogenerator/internal/policy/parser"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/tester"
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
    } with input.attributes.request.http as {
        "path": "/bearer",
        "method": "get",
    }
}`

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
	minioRepo, err := bundle.NewMinioRepositoryFromConfig()
	if err != nil {
		return fmt.Errorf("error creating minio repository: %w", err)
	}
	// Create the bucket if it does not exist
	if err := minioRepo.CreateBucket(ctx); err != nil {
		return fmt.Errorf("error creating bucket: %w", err)
	}
	minioRepo.CreateBucket(ctx)
	slog.Info("Connected to MinIO successfully")

	// Verify if the bundle exists on minio
	bundleExists, err := minioRepo.BundleExists(ctx, config.LatestBundleName)
	if err != nil {
		return fmt.Errorf("error checking if service exists: %w", err)
	}

	// If the bundle does not exist, create it
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
		options := generator.ServiceOptions{
			ServiceName: serviceName,
			PathPrefix:  "/httpbin",
		}
		err = generator.GenerateServiceFolder(options, regoDir, *provider, policies)
		if err != nil {
			return fmt.Errorf("error generating service folder: %w", err)
		}

		serviceList := append(generator.StaticServiceNames, serviceName)

		err = generator.GenerateNewMain(regoDir, serviceList)
		if err != nil {
			return fmt.Errorf("error generating main.rego: %w", err)
		}

		// Write the test file
		testFilePath := filepath.Join(regoDir, serviceName, "test_test.rego")
		err = os.WriteFile(testFilePath, []byte(testFile), 0644)
		if err != nil {
			return fmt.Errorf("error writing test file: %w", err)
		}

		// Build the bundle
		b, err := bundle.NewFromFS(ctx, os.DirFS(tempDir), serviceList...)
		if err != nil {
			return fmt.Errorf("error building bundle: %w", err)
		}
		if err := minioRepo.Write(config.LatestBundleName, *b); err != nil {
			return fmt.Errorf("error writing bundle to Minio: %w", err)
		}
		slog.Info("Bundle written to MinIO successfully")
	}

	// Download the bundle from MinIO
	bundlePath, err := minioRepo.Read(config.LatestBundleName)
	if err != nil {
		return fmt.Errorf("error downloading bundle from MinIO: %w", err)
	}
	testBundlePath := filepath.Join(os.TempDir(), config.LatestBundleName)
	fsRepo := bundle.NewFileSystemRepository(filepath.Dir(testBundlePath))
	if err := fsRepo.Write(config.LatestBundleName, *bundlePath); err != nil {
		return fmt.Errorf("error writing bundle to file: %w", err)
	}

	// Load the bundle and execute the tests
	bundles, err := tester.LoadBundlesWithRegoVersion([]string{testBundlePath}, func(abspath string, info fs.FileInfo, depth int) bool {
		return true
	}, ast.DefaultRegoVersion)
	if err != nil {
		return err
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
