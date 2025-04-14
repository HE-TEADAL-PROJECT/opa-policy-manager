package main

import (
	"fmt"
	"os"
)

// This function load the bundle from the specified URI and returns the bundle data.
// It returns an error if the bundle cannot be loaded.
// The uri can specified a local file or a file stored in Minio.
func loadBundle(bundleURI string) ([]byte, error) {
	remoteBundle := false
	if bundleURI == "" {
		return nil, fmt.Errorf("bundle URI is empty")
	}
	if bundleURI[:4] == "http" || bundleURI[:5] == "https" {
		remoteBundle = true
	}

	if remoteBundle {
		//TODO(bramba2000): implement remote bundle loading
		return nil, fmt.Errorf("remote bundle loading is not supported yet")
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %v", err)
		}
		fmt.Printf("Loading bundle from local file (cwd %s): %s\n", cwd, bundleURI)
		file, err := os.Open(bundleURI)
		if err != nil {
			return nil, fmt.Errorf("failed to open bundle file: %v", err)
		}
		defer file.Close()
		bundleData, err := os.ReadFile(bundleURI)
		if err != nil {
			return nil, fmt.Errorf("failed to read bundle file: %v", err)
		}
		return bundleData, nil
	}
}
