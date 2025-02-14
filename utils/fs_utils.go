package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func RemoveFilesInDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return err
		}
	}
	return nil
}

func ReplacePlaceholdersInFile(myfile string, replacements map[string]string) error {

	content, err := os.ReadFile(myfile)
	if err != nil {
		return fmt.Errorf("error reading input file %s: %v", myfile, err)
	}

	updatedContent := string(content)
	for placeholder, value := range replacements {
		updatedContent = strings.ReplaceAll(updatedContent, placeholder, value)
	}

	if err := os.WriteFile(myfile, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("error writing output file: %v", err)
	}

	return nil
}

func ExtractServerName(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}
	return parsedURL.Host, nil
}

func UniqueElements(input map[string][]interface{}) []string {
	uniqueMap := make(map[string]struct{})
	var result []string

	for _, values := range input {
		for _, v := range values {
			if str, ok := v.(string); ok {
				if _, exists := uniqueMap[str]; !exists {
					uniqueMap[str] = struct{}{}
					result = append(result, str)
				}
			}
		}
	}

	return result
}

func GetMethodType(http_method string) string {
	read := []string{"get", "head", "options"}
	write := []string{"put", "patch", "post", "delete"}
	if slices.Contains(read, strings.ToLower(http_method)) {
		return "read"
	}
	if slices.Contains(write, strings.ToLower(http_method)) {
		return "write"
	}
	return ""
}

func ExtractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
		}
	}

	return nil
}

func MoveDirectory(inputDir, outputDir string) error {
	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get the final destination path (outputDir/input)
	destinationPath := filepath.Join(outputDir, filepath.Base(inputDir))

	// Move the directory
	if err := os.Rename(inputDir, destinationPath); err != nil {
		fmt.Errorf("the same directory exists: %w", err)
		if err := os.RemoveAll(destinationPath); err != nil {
			return fmt.Errorf("can't delete the existing dir: %w", err)
		} else {
			if err := os.Rename(inputDir, destinationPath); err != nil {
				fmt.Errorf("cannot move the dir: %w", err)
			}
		}
	}

	return nil
}

func DownloadFile(url, filePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to initiate request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: status code %d", resp.StatusCode)
	}

	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}
