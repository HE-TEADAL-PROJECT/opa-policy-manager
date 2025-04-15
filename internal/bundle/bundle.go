package bundle

import (
	"context"
	"dspn-regogenerator/config"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/compile"
)

func BuildBundle(bundleDir, outputDir string) (*bundle.Bundle, error) {
	// Create a new compiler
	compiler := compile.New().WithAsBundle(true).WithPaths(bundleDir)

	// Compile the directory
	if err := compiler.Build(context.Background()); err != nil {
		log.Fatalf("Failed to compile: %v", err)
	}

	// Access the compiled bundle
	return compiler.Bundle(), nil
}

func WriteBundleToFile(b *bundle.Bundle, outputDir string) error {
	// Write the bundle to a file
	file, err := os.OpenFile(outputDir, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("write bundle: failed to open file %w", err)
	}
	bundle.NewWriter(file).Write(*b)
	return nil
}

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

func WriteBundleToMinio(b *bundle.Bundle) error {
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

	if _, err := client.PutObject(context.Background(), config.Config.Bucket_Name, config.Config.BundleFileName, reader, -1, minio.PutObjectOptions{}); err != nil {
		return fmt.Errorf("write bundle: failed to upload to MinIO %w", err)
	}

	return nil
}

func LoadBundleFromMinio() (*bundle.Bundle, error) {
	client, err := minio.New(config.Config.Minio_Server, &minio.Options{
		Creds: credentials.NewStaticV4(config.Config.Minio_Access_Key, config.Config.Minio_Secret_Key, "")})
	if err != nil {
		return nil, fmt.Errorf("load bundle: failed to create minio client %w", err)
	}

	// Get the object from MinIO
	object, err := client.GetObject(context.Background(), config.Config.Bucket_Name, config.Config.BundleFileName, minio.GetObjectOptions{})
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
