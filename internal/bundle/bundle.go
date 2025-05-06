package bundle

import (
	"context"
	"dspn-regogenerator/internal/config"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hashicorp/go-set/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/compile"
	"github.com/open-policy-agent/opa/v1/loader"
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

func getServiceSet(rootDir string, bundlePaths []string) *set.Set[string] {
	serviceSet := set.New[string](len(bundlePaths))
	for _, bundlePath := range bundlePaths {
		// Check if the bundlePath is a directory
		if info, err := os.Stat(filepath.Join(rootDir, bundlePath)); err == nil && info.IsDir() {
			serviceSet.Insert(info.Name())
		} else if err == nil && !info.IsDir() {
			// If it's not a directory, the service name is either:
			// - the directory name (if the file is in a subdir)
			// - the file name without extension (if the file is in the root dir)
			serviceName := strings.Split(filepath.Base(bundlePath), ".")[0]
			if strings.Contains(bundlePath, "/") {
				serviceName = filepath.Dir(bundlePath)
			}
			serviceSet.Insert(serviceName)
		} else {
			fmt.Printf("getServiceSet: failed to stat %s: %v\n", bundlePath, err)
		}
	}
	return serviceSet
}

// CompileBundle compiles the bundle from the given directory and returns it as a bundle object.
// The provided list of bundle paths is used to specify which are the services roots: if it is a folder, all recursive files will be included, otherwise, the single file will be included as a standalone service module.
// All provided bundles should be relative to the rootDir.
// If no bundle paths are provided, it defaults to the current directory (".").
func CompileBundle(ctx context.Context, rootDir string, bundlePaths ...string) (*bundle.Bundle, error) {
	// Load service list
	serviceSet := getServiceSet(rootDir, bundlePaths)

	if len(bundlePaths) == 0 {
		bundlePaths = []string{"."}
	}
	fs := os.DirFS(rootDir)
	compiler := compile.New().WithFS(fs).WithPaths(bundlePaths...)

	if err := compiler.Build(ctx); err != nil {
		return nil, fmt.Errorf("compile bundle: failed to compile %s: %w", strings.Join(bundlePaths, ","), err)
	}
	b := compiler.Bundle()
	b.Manifest.Metadata = map[string]interface{}{
		"services": *serviceSet,
	}

	return b, nil
}

func AddServiceFolder(originalBundle *bundle.Bundle, rootFolder string, serviceFolders ...string) (*bundle.Bundle, error) {
	newBundle := originalBundle.Copy()

	if len(serviceFolders) == 0 {
		serviceFolders = []string{"."}
	}

	result, err := loader.NewFileLoader().WithFS(os.DirFS(rootFolder)).All(serviceFolders)
	if err != nil {
		return nil, fmt.Errorf("add service folder: failed to load files %s: %w", strings.Join(serviceFolders, ","), err)
	}

	// Delete all existing modules whose path start with a service folder
	for _, serviceFolder := range serviceFolders {
		newBundle.Modules = slices.DeleteFunc(newBundle.Modules, func(m bundle.ModuleFile) bool {
			return strings.HasPrefix(m.Path, serviceFolder)
		})
	}

	modulesPositionsMap := make(map[string]int)
	for i, mod := range newBundle.Modules {
		modulesPositionsMap[mod.Path] = i
	}

	for _, file := range result.Modules {
		if _, ok := modulesPositionsMap[file.Name]; ok {
			panic("Module should be deleted before")
		} else {
			// If the module doesn't exist, add it
			newBundle.Modules = append(newBundle.Modules, bundle.ModuleFile{
				URL:    file.Name,
				Path:   file.Name,
				Raw:    file.Raw,
				Parsed: file.Parsed,
			})
		}
	}

	var currentServices set.Set[string]
	if services, ok := newBundle.Manifest.Metadata["services"]; ok {
		if s, ok := services.(set.Set[string]); ok {
			currentServices = s
		} else if s, ok := services.([]string); ok {
			currentServices = *set.From(s)
		} else {
			return nil, fmt.Errorf("add service folder: failed to parse services metadata")
		}
	} else {
		newBundle.Manifest.Metadata = map[string]interface{}{}
		currentServices = *set.New[string](len(serviceFolders))
	}
	// Add the new service folders to the bundle metadata
	newServiceSet := getServiceSet(rootFolder, serviceFolders)
	// Set the updated services metadata
	newBundle.Manifest.Metadata["services"] = *currentServices.Union(newServiceSet).(*set.Set[string])

	return &newBundle, nil
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
	client, err := minio.New(config.MinioEndpoint, &minio.Options{
		Creds: credentials.NewStaticV4(config.MinioAccessKey, config.MinioSecretKey, "")})
	if err != nil {
		return fmt.Errorf("write bundle: failed to create minio client %w", err)
	}
	// Create a new bucket if it doesn't exist
	err = client.MakeBucket(context.Background(), config.MinioBucket, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := client.BucketExists(context.Background(), config.MinioBucket)
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
		bundleWriter := bundle.NewWriter(writer).UseModulePath(true)
		if err := bundleWriter.Write(*b); err != nil {
			fmt.Fprintf(os.Stderr, "write bundle: failed to write bundle to pipe %v\n", err)
		}
	}()

	if _, err := client.PutObject(context.Background(), config.MinioBucket, bundleFileName, reader, -1, minio.PutObjectOptions{}); err != nil {
		return fmt.Errorf("write bundle: failed to upload to MinIO %w", err)
	}

	return nil
}

// LoadBundleFromMinio loads the bundle from MinIO using the MinIO client.
// It retrieves the bundle file from the specified bucket and returns it as a bundle object.
func LoadBundleFromMinio(bundleFileName string) (*bundle.Bundle, error) {
	client, err := minio.New(config.MinioEndpoint, &minio.Options{
		Creds: credentials.NewStaticV4(config.MinioAccessKey, config.MinioSecretKey, "")})
	if err != nil {
		return nil, fmt.Errorf("load bundle: failed to create minio client %w", err)
	}

	// Get the object from MinIO
	object, err := client.GetObject(context.Background(), config.MinioBucket, bundleFileName, minio.GetObjectOptions{})
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

func CheckBundleFileExists(bundleFileName string) (bool, error) {
	// Check if the file exists
	client, err := minio.New(config.MinioEndpoint, &minio.Options{
		Creds: credentials.NewStaticV4(config.MinioAccessKey, config.MinioSecretKey, "")})
	if err != nil {
		return false, fmt.Errorf("load bundle: failed to create minio client %w", err)
	}

	// Check if the object exists in the bucket
	exists, err := client.BucketExists(context.Background(), config.MinioBucket)
	if err != nil {
		return false, fmt.Errorf("load bundle: failed to check if bucket exists %w", err)
	}
	if !exists {
		return false, fmt.Errorf("load bundle: bucket %s does not exist", config.MinioBucket)
	}
	// Check if the object exists in the bucket
	objectInfo, err := client.StatObject(context.Background(), config.MinioBucket, bundleFileName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil // Object does not exist
		}
		return false, fmt.Errorf("load bundle: failed to check if object exists %w", err)
	}
	if objectInfo.Size == 0 {
		return false, fmt.Errorf("load bundle: object %s is empty", bundleFileName)
	}
	return true, nil // Object exists
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

// AddRegoFilesFromDirectory creates a new bundle with all content from the original bundle
// plus Rego files loaded from the specified directory path
func AddRegoFilesFromDirectory(originalBundle *bundle.Bundle, bundleRootDir string) (*bundle.Bundle, error) {
	// Create a copy of the existing bundle
	newBundle := originalBundle.Copy()

	// Ensure the manifest is initialized
	newBundle.Manifest.Init()

	// Get the parser options based on the bundle's Rego version
	parserOpts := ast.ParserOptions{
		ProcessAnnotation: true,
		Capabilities:      ast.CapabilitiesForThisVersion(),
		RegoVersion:       newBundle.RegoVersion(ast.DefaultRegoVersion),
	}

	// Track the directories we find Rego files in
	regoDirs := make(map[string]struct{})

	// Create a map to track existing modules by their normalized paths
	existingModules := make(map[string]int)
	for i, mod := range newBundle.Modules {
		// Normalize path by removing leading slash if present
		normalizedModPath := strings.TrimPrefix(mod.Path, "/")
		existingModules[normalizedModPath] = i
	}

	// Create a list to store new modules we're adding
	var newModules []bundle.ModuleFile

	// Walk through the directory and process each .rego file
	err := filepath.Walk(bundleRootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-rego files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), bundle.RegoExt) {
			return nil
		}

		// Read the file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Calculate relative path within the bundle
		relPath, err := filepath.Rel(bundleRootDir, path)
		if err != nil {
			return fmt.Errorf("failed to determine relative path for %s: %w", path, err)
		}

		// Normalize the path for OPA bundle (use forward slashes)
		normalizedPath := filepath.ToSlash(relPath)

		// Parse the Rego module
		parsedModule, err := ast.ParseModuleWithOpts(normalizedPath, string(content), parserOpts)
		if err != nil {
			return fmt.Errorf("failed to parse module %s: %w", normalizedPath, err)
		}

		// Create the ModuleFile
		moduleFile := bundle.ModuleFile{
			Path:         normalizedPath, // Keep the original form for new modules
			URL:          normalizedPath,
			RelativePath: normalizedPath,
			Raw:          content,
			Parsed:       parsedModule,
		}

		// Check if this module already exists in the original bundle
		normalizedCheckPath := normalizedPath // No need to trim prefix as loaded paths don't have leading slash
		if idx, exists := existingModules[normalizedCheckPath]; exists {
			// Replace the existing module with the new one
			// But preserve the original path format (with or without leading slash)
			originalPath := newBundle.Modules[idx].Path
			moduleFile.Path = originalPath
			moduleFile.URL = originalPath

			newBundle.Modules[idx] = moduleFile
			// Mark as processed so we don't add it again later
			delete(existingModules, normalizedCheckPath)
		} else {
			// This is a new module, add it to our new modules list
			newModules = append(newModules, moduleFile)
		}

		// Remember this directory for adding to roots
		moduleDir := filepath.Dir(normalizedPath)
		if moduleDir != "" && moduleDir != "." {
			regoDirs[strings.TrimSuffix(moduleDir, "/")] = struct{}{}
		}

		return nil
	})

	if err != nil {
		return &newBundle, fmt.Errorf("error walking directory %s: %w", bundleRootDir, err)
	}

	// Add any new modules to the bundle
	if len(newModules) > 0 {
		newBundle.Modules = append(newBundle.Modules, newModules...)
	}

	// Add each directory containing Rego files to the bundle roots if not already included
	for dir := range regoDirs {
		// Check if this directory is already covered by an existing root
		alreadyCovered := false
		for _, root := range *newBundle.Manifest.Roots {
			if bundle.RootPathsOverlap(root, dir) {
				alreadyCovered = true
				break
			}
		}

		if !alreadyCovered {
			newBundle.Manifest.AddRoot(dir)
		}
	}

	// For debugging - print replaced modules
	// TODO: fix wrong counting of modules

	return &newBundle, nil
}

// RemoveService creates a new bundle without all files that belongs to a service subdir (/<mainDir>/<subdir>)
func RemoveService(originalBundle *bundle.Bundle, subdir string) (*bundle.Bundle, error) {
	// Extract the main directory from the original bundle
	mainDir, ok := originalBundle.Manifest.Metadata["main"].(string)
	if !ok {
		return nil, fmt.Errorf("remove service: failed to find main directory in bundle manifest")
	}

	// Create a copy of the existing bundle
	newBundle := originalBundle.Copy()

	// Normalize the subdir name by removing any leading/trailing slashes
	subdir = strings.Trim(subdir, "/")

	// The pattern we're looking for - both with and without leading slash
	pattern := fmt.Sprintf("%s/%s/", mainDir, subdir)
	patternWithSlash := fmt.Sprintf("/%s/%s/", mainDir, subdir)

	// Filter out modules that match the pattern
	var filteredModules []bundle.ModuleFile
	for _, mod := range newBundle.Modules {
		// Check both with and without leading slash
		if !strings.HasPrefix(mod.Path, pattern) && !strings.HasPrefix(mod.Path, patternWithSlash) {
			filteredModules = append(filteredModules, mod)
		}
	}

	// Replace the modules with the filtered list
	newBundle.Modules = filteredModules

	// Check if the removed modules had a dedicated root that's no longer needed
	regoRoot := fmt.Sprintf("rego/%s", subdir)

	// Only modify roots if they exist
	if newBundle.Manifest.Roots != nil {
		var updatedRoots []string
		for _, root := range *newBundle.Manifest.Roots {
			// Keep all roots except the one specific to this subdir
			trimmedRoot := strings.Trim(root, "/")
			if trimmedRoot != regoRoot {
				updatedRoots = append(updatedRoots, root)
			}
		}
		newBundle.Manifest.Roots = &updatedRoots
	}

	fmt.Printf("Removed modules matching pattern 'rego/%s/' - bundle now contains %d modules\n",
		subdir, len(newBundle.Modules))

	return &newBundle, nil
}

func RenameBundleFileName(oldFileName, newFileName string) error {
	// Create a copy of the existing bundle
	client, err := minio.New(config.MinioEndpoint, &minio.Options{
		Creds: credentials.NewStaticV4(config.MinioAccessKey, config.MinioSecretKey, "")})
	if err != nil {
		return fmt.Errorf("load bundle: failed to create minio client %w", err)
	}

	if _, err = client.CopyObject(context.TODO(), minio.CopyDestOptions{
		Bucket: config.MinioBucket,
		Object: newFileName,
	}, minio.CopySrcOptions{
		Bucket: config.MinioBucket,
		Object: oldFileName,
	}); err != nil {
		return fmt.Errorf("load bundle: failed to copy object %w", err)
	}

	return nil
}
