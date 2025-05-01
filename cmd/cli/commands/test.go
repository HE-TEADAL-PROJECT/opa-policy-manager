package commands

import (
	"context"
	"dspn-regogenerator/internal/bundle"
	"dspn-regogenerator/internal/config"
	"dspn-regogenerator/internal/generator"
	"dspn-regogenerator/internal/policy/parser"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/tester"
	"github.com/spf13/cobra"
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

var (
	testBundlePath string
)

var TestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the bundle",
	Long:  `Test the bundle against the httpbin service. This will run all the tests in the bundle and print the results.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		serviceName := "testBundle"
		// testPath := "./testdata/rego/oidc_test.rego"
		testSchemaPath := "./testdata/schemas/httpbin-api.json"

		// Print the current configuration
		cmd.Println("Configuration:")
		cmd.Println("    MinIO Endpoint:", config.MinioEndpoint)
		cmd.Println("    MinIO Access Key:", config.MinioAccessKey)
		cmd.Println("    MinIO Secret Key:", config.MinioSecretKey)
		cmd.Println("    MinIO Bucket:", config.MinioBucket)
		cmd.Println("    MinIO Bundle Prefix:", config.MinioBundlePrefix)
		cmd.Println("    MinIO Timeout:", config.MinioTimeout)

		// Test the MinIO connection
		if err := testMinioConnection(ctx); err != nil {
			cmd.Println("Error connecting to MinIO:", err)
			return
		}
		cmd.Println("MinIO connection successful and bucket exists or created.")

		// Verify if the bundle exists on minio
		bundleExists, err := bundle.CheckBundleFileExists(config.LatestBundleName)
		if err != nil {
			cmd.PrintErrf("Error checking if service exists: %v\n", err)
			return
		}
		if !bundleExists {

			// Load the OpenAPI spec file
			specData, err := loadSpecFile(testSchemaPath)
			if err != nil {
				cmd.PrintErrf("Error loading OpenAPI spec file: %v\n", err)
				return
			}

			// Parse the OpenAPI spec to extract policies and provider
			policies, err := parser.ParseOpenAPIPolicies(specData)
			if err != nil || policies == nil {
				cmd.PrintErrf("Error parsing OpenAPI spec: %v\n", err)
				return
			}
			provider, err := parser.ParseOpenAPIIAM(specData)
			if err != nil || provider == nil {
				cmd.PrintErrf("Error parsing OpenAPI provider: %v\n", err)
				return
			}

			// Create a temporary directory for the output
			tempDir, err := os.MkdirTemp("", "bundle-*")
			if err != nil {
				cmd.PrintErrf("Error creating temp directory: %v\n", err)
				return
			}
			regoDir := filepath.Join(tempDir, "rego")
			err = os.MkdirAll(regoDir, os.ModePerm)
			if err != nil {
				cmd.PrintErrf("Error creating rego directory: %v\n", err)
				return
			}

			// Generate the service folder
			err = generator.GenerateServiceFolder(serviceName, regoDir, *provider, policies)
			if err != nil {
				cmd.PrintErrf("Error generating service folder: %v\n", err)
				return
			}

			// Write the test file
			testFilePath := filepath.Join(regoDir, serviceName, "test_test.rego")
			err = os.WriteFile(testFilePath, []byte(testFile), 0644)
			if err != nil {
				cmd.PrintErrf("Error writing test file: %v\n", err)
				return
			}

			// Build the bundle
			b, err := bundle.BuildBundle(tempDir, "rego")
			if err != nil {
				cmd.PrintErrf("Error building bundle: %v\n", err)
				return
			}
			if err != nil {
				cmd.PrintErrf("Error building bundle: %v\n", err)
				return
			}
			if err := bundle.WriteBundleToMinio(b, config.LatestBundleName); err != nil {
				cmd.PrintErrf("Error writing bundle to Minio: %v\n", err)
				return
			}
			cmd.Printf("Bundle created and uploaded successfully, a copy is available at %s\n", tempDir)
		}

		// Download the bundle from MinIO
		bundlePath, err := bundle.LoadBundleFromMinio(config.LatestBundleName)
		if err != nil {
			cmd.PrintErrf("Error downloading bundle from MinIO: %v\n", err)
		}
		testBundlePath = filepath.Join(os.TempDir(), config.LatestBundleName)
		err = bundle.WriteBundleToFile(bundlePath, testBundlePath)
		if err != nil {
			cmd.PrintErrf("Error writing bundle to file: %v\n", err)
			return
		}

		// Load the bundle and execute the tests
		bundles, err := tester.LoadBundles([]string{testBundlePath}, nil)
		if err != nil {
			cmd.Println("Error loading bundle:", err)
			return
		}
		store := inmem.New()

		testRunner := tester.NewRunner()
		testRunner.SetBundles(bundles)
		testRunner.SetStore(store)
		ch, err := testRunner.RunTests(ctx, storage.NewTransactionOrDie(ctx, store, storage.WriteParams))
		if err != nil {
			cmd.Println("Error running tests:", err)
			return
		}
		for result := range ch {
			if result.Error != nil || result.Fail {
				cmd.Println(result.Name, "failed:", result.Error)
			} else {
				cmd.Println(result.Name, "passed")
			}
		}
	},
}

func init() {
	TestCmd.Flags().StringVarP(&testBundlePath, "bundle-path", "b", "./testdata/rego/", "Path to the bundle or dir to test")
}
