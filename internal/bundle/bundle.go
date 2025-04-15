package bundle

import (
	"context"
	"fmt"
	"log"
	"os"

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
