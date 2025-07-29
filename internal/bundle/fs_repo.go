package bundle

import (
	"os"

	opabundle "github.com/open-policy-agent/opa/v1/bundle"
)

type FSRepository struct{}

// Get implements Repository.
func (f FSRepository) Get(path string) (*Bundle, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return NewFromTarball(reader)
}

// Save implements Repository.
func (f FSRepository) Save(path string, bundle Bundle) error {
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	opabundle.NewWriter(w).Write(*bundle.bundle)
	return nil
}

var _ Repository = FSRepository{}
