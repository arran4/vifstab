package skill

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// LocalSource handles fetching skills from a local directory path.
type LocalSource struct {
	Path string
}

func (s *LocalSource) FetchTo(destDir string) (string, error) {
	// check if source path exists and is a dir
	info, err := os.Stat(s.Path)
	if err != nil {
		return "", fmt.Errorf("local source not found: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("local source %s is not a directory", s.Path)
	}

	err = copyDir(s.Path, destDir)
	if err != nil {
		return "", fmt.Errorf("failed to copy local source: %w", err)
	}

	// local hash/revision can be a hash of the files
	return "local-copy", nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Copy file securely
		return copyFile(path, destPath)
	})
}

func copyFile(srcFile, dstFile string) error {
	data, err := os.ReadFile(srcFile)
	if err != nil {
		return err
	}
	// Do not preserve execute permissions for skills
	return os.WriteFile(dstFile, data, 0644)
}
