package publish

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Allowlisted directories and files to include in the source tarball.
var allowedDirs = []string{
	"pages",
	"components",
	"composables",
	"engine",
	"agent",
	"utils",
	"types",
	"views",
	"stores",
	"spaces",
	"data",
	"src",
	"public",
	"tests",
	"widgets",
}

var allowedRootFiles = []string{
	"space.manifest.json",
	"package.json",
	"tsconfig.json",
	"vite.config.ts",
	"vite.config.js",
	"space.config.ts",
	"types.ts",
	"index.ts",
}

// Root file patterns that are included via glob matching (e.g. *.config.ts)
var allowedRootPatterns = []string{
	"*.config.ts",
	"*.config.js",
}

// Blocked extensions that should never be included
var blockedExtensions = []string{
	".env",
	".log",
	".lock",
	".lockb",
}

const maxTarballSize = 50 * 1024 * 1024 // 50MB

// PackSource creates a .tar.gz of the space source code using an allowlist approach.
// Returns the path to the temporary tarball file.
func PackSource(root string) (string, error) {
	tmpFile, err := os.CreateTemp("", "space-source-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tarballPath := tmpFile.Name()

	gzWriter := gzip.NewWriter(tmpFile)
	tarWriter := tar.NewWriter(gzWriter)

	var totalSize int64

	// Add allowed root files
	for _, name := range allowedRootFiles {
		path := filepath.Join(root, name)
		info, err := os.Stat(path)
		if err != nil {
			continue // file doesn't exist, skip
		}
		if err := addFileToTar(tarWriter, root, path, info); err != nil {
			tarWriter.Close()
			gzWriter.Close()
			tmpFile.Close()
			os.Remove(tarballPath)
			return "", fmt.Errorf("failed to add %s: %w", name, err)
		}
		totalSize += info.Size()
	}

	// Add root files matching allowed patterns
	rootEntries, _ := os.ReadDir(root)
	for _, entry := range rootEntries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip if already added as an explicit root file
		alreadyAdded := false
		for _, rf := range allowedRootFiles {
			if rf == name {
				alreadyAdded = true
				break
			}
		}
		if alreadyAdded {
			continue
		}
		for _, pattern := range allowedRootPatterns {
			if matched, _ := filepath.Match(pattern, name); matched {
				path := filepath.Join(root, name)
				info, err := os.Stat(path)
				if err != nil {
					break
				}
				if err := addFileToTar(tarWriter, root, path, info); err != nil {
					tarWriter.Close()
					gzWriter.Close()
					tmpFile.Close()
					os.Remove(tarballPath)
					return "", fmt.Errorf("failed to add %s: %w", name, err)
				}
				totalSize += info.Size()
				break
			}
		}
	}

	// Add allowed directories recursively
	for _, dir := range allowedDirs {
		dirPath := filepath.Join(root, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip errors
			}

			// Skip hidden files/dirs
			if strings.HasPrefix(info.Name(), ".") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip node_modules and dist inside any directory
			if info.IsDir() && (info.Name() == "node_modules" || info.Name() == "dist") {
				return filepath.SkipDir
			}

			// Skip blocked extensions
			for _, ext := range blockedExtensions {
				if strings.HasSuffix(info.Name(), ext) {
					return nil
				}
			}

			if info.IsDir() {
				return nil
			}

			totalSize += info.Size()
			if totalSize > maxTarballSize {
				return fmt.Errorf("source exceeds maximum size of %dMB", maxTarballSize/1024/1024)
			}

			return addFileToTar(tarWriter, root, path, info)
		})

		if err != nil {
			tarWriter.Close()
			gzWriter.Close()
			tmpFile.Close()
			os.Remove(tarballPath)
			return "", err
		}
	}

	if err := tarWriter.Close(); err != nil {
		gzWriter.Close()
		tmpFile.Close()
		os.Remove(tarballPath)
		return "", err
	}
	if err := gzWriter.Close(); err != nil {
		tmpFile.Close()
		os.Remove(tarballPath)
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tarballPath)
		return "", err
	}

	return tarballPath, nil
}

func addFileToTar(tw *tar.Writer, root, path string, info os.FileInfo) error {
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = relPath

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(tw, f)
	return err
}
