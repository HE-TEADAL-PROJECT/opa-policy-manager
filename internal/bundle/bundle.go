package bundle

import (
	"context"
	"dspn-regogenerator/config"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/compile"
)

// BuildBundle compiles the bundle from the given directory and returns it as a bundle object.
// It uses the OPA compiler to compile the bundle and returns the compiled bundle.
func BuildBundle(bundleDir string, mainDir string) (*bundle.Bundle, error) {
	// Create a new compiler
	compiler := compile.New().WithAsBundle(true).WithFS(os.DirFS(bundleDir)).WithPaths(mainDir).WithMetadata(&map[string]interface{}{
		"main": mainDir,
	}).WithRoots(mainDir)

	// Compile the directory
	if err := compiler.Build(context.Background()); err != nil {
		return nil, fmt.Errorf("build bundle: failed to compile %s (mainDir %s): %w", bundleDir, mainDir, err)
	}

	// Access the compiled bundle
	return compiler.Bundle(), nil
}

// WriteBundleToFile writes the bundle to a file in the specified output file path.
// It overwrites the file if it already exists, truncating it to zero length.
func WriteBundleToFile(b *bundle.Bundle, outputFilePath string) error {
	// Write the bundle to a file
	file, err := os.OpenFile(outputFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("write bundle: failed to open file %w", err)
	}
	bundle.NewWriter(file).UseModulePath(true).Write(*b)
	return nil
}

// LoadBundleFromFile reads the bundle from the specified file path and returns it as a bundle object.
func LoadBundleFromFile(filePath string) (*bundle.Bundle, error) {
	// Open the bundle file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("load bundle: failed to open file %w", err)
	}
	defer file.Close()

	// Read the bundle from the file
	bundleReader := bundle.NewReader(file)
	b, err := bundleReader.Read()
	if err != nil {
		return nil, fmt.Errorf("load bundle: failed to read bundle %w", err)
	}

	return &b, nil
}

// WriteBundleToMinio writes the bundle to MinIO using the MinIO client.
// It creates a new bucket if it doesn't exist and uploads the bundle to the specified bucket.
// The configuration for the MinIO server, access key, secret key, and bucket name is taken from the config package.
func WriteBundleToMinio(b *bundle.Bundle, bundleFileName string) error {
	client, err := minio.New(config.Config.Minio_Server, &minio.Options{
		Creds: credentials.NewStaticV4(config.Config.Minio_Access_Key, config.Config.Minio_Secret_Key, "")})
	if err != nil {
		return fmt.Errorf("write bundle: failed to create minio client %w", err)
	}
	// Create a new bucket if it doesn't exist
	err = client.MakeBucket(context.Background(), config.Config.Bucket_Name, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := client.BucketExists(context.Background(), config.Config.Bucket_Name)
		if errBucketExists != nil {
			return fmt.Errorf("write bundle: failed to check if bucket exists %w", errBucketExists)
		}
		if !exists {
			return fmt.Errorf("write bundle: failed to create bucket %w", err)
		}
	}

	// Create a pipe to write the bundle to MinIO
	reader, writer := io.Pipe()

	// Start a goroutine to write the bundle to the pipe
	go func() {
		defer writer.Close()
		bundleWriter := bundle.NewWriter(writer)
		if err := bundleWriter.Write(*b); err != nil {
			fmt.Fprintf(os.Stderr, "write bundle: failed to write bundle to pipe %v\n", err)
		}
	}()

	if _, err := client.PutObject(context.Background(), config.Config.Bucket_Name, bundleFileName, reader, -1, minio.PutObjectOptions{}); err != nil {
		return fmt.Errorf("write bundle: failed to upload to MinIO %w", err)
	}

	return nil
}

// LoadBundleFromMinio loads the bundle from MinIO using the MinIO client.
// It retrieves the bundle file from the specified bucket and returns it as a bundle object.
func LoadBundleFromMinio(bundleFileName string) (*bundle.Bundle, error) {
	client, err := minio.New(config.Config.Minio_Server, &minio.Options{
		Creds: credentials.NewStaticV4(config.Config.Minio_Access_Key, config.Config.Minio_Secret_Key, "")})
	if err != nil {
		return nil, fmt.Errorf("load bundle: failed to create minio client %w", err)
	}

	// Get the object from MinIO
	object, err := client.GetObject(context.Background(), config.Config.Bucket_Name, bundleFileName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("load bundle: failed to get object from MinIO %w", err)
	}
	defer object.Close()

	bundleReader := bundle.NewReader(object)
	b, err := bundleReader.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load bundle: failed to read bundle \n%v", object)
		return nil, fmt.Errorf("load bundle: failed to read %w", err)
	}

	return &b, nil
}

func ListBundleFiles(b *bundle.Bundle) []string {
	dirs := make([]string, 0)
	mainDir, ok := b.Manifest.Metadata["main"].(string)
	if !ok {
		for _, mod := range b.Modules {
			dirs = append(dirs, mod.Path)
		}
		return dirs
	}
	// List the directories in the bundle
	for _, mod := range b.Modules {
		dirs = append(dirs, strings.Split(mod.Path, mainDir+"/")[1])
	}
	return dirs
}
