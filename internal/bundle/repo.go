package bundle

// BundleRepository is an interface for writing bundle to a storage system.
type BundleRepository interface {
	// Write a bundle to the repository, returning an error if it fails.
	Write(path string, bundle Bundle) error

	// Read reads the bundle from the repository, returning the bundle and an error if it fails.
	Read(path string) (*Bundle, error)
}
