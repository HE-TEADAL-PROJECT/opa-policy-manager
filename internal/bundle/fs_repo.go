package bundle

import (
	"context"
	"os"
	"path/filepath"

	opabundle "github.com/open-policy-agent/opa/v1/bundle"
)

type FileSystemBundleRepository struct {
	// Path to base directory
	basePath string
}

// Read implements BundleRepository.
func (f *FileSystemBundleRepository) Read(path string) (*Bundle, error) {
	fullPath := filepath.Join(f.basePath, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	bundle, err := NewFromArchive(context.TODO(), file)
	if err != nil {
		return nil, err
	}
	return bundle, nil
}

// Write implements BundleRepository.
func (f *FileSystemBundleRepository) Write(path string, bundle Bundle) error {
	fullPath := filepath.Join(f.basePath, path)
	fullDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return err
	}
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := opabundle.NewWriter(file)
	if err := writer.Write(*bundle.bundle); err != nil {
		return err
	}
	return nil
}

func NewFileSystemBundleRepository(baseDir string) *FileSystemBundleRepository {
	return &FileSystemBundleRepository{
		basePath: baseDir,
	}
}

var _ BundleRepository = (*FileSystemBundleRepository)(nil)
