package editor

import (
	"fmt"
	"os"
	"path/filepath"
)

// effectiveTarget identifies the regular file to replace, rather than the
// configured path's final symlink. Missing regular files may be created, but
// dangling links and missing parent directories are never implicitly repaired.
// Resolution is injectable and shared by destination metadata and replacement.
type effectiveTarget struct {
	path string
	info os.FileInfo // nil only for a new file
}

func resolveTarget(name string, fs FS) (effectiveTarget, error) {
	_, err := fs.Lstat(name)
	if err != nil && !os.IsNotExist(err) {
		return effectiveTarget{}, fmt.Errorf("failed to stat destination file: %w", err)
	}
	var resolved string
	var info os.FileInfo
	if err == nil {
		resolved, err = fs.EvalSymlinks(name)
		if err != nil {
			return effectiveTarget{}, fmt.Errorf("failed to resolve destination; check link targets and permissions: %w", err)
		}
		info, err = fs.Stat(resolved)
		if err != nil {
			return effectiveTarget{}, fmt.Errorf("failed to stat resolved destination file: %w", err)
		}
		if !info.Mode().IsRegular() {
			return effectiveTarget{}, fmt.Errorf("destination must resolve to a regular file")
		}
	} else {
		// Split without cleaning first: link/../new must resolve link before ..
		// just as the kernel does. Cleaning the original path changes its meaning.
		dir, base := filepath.Split(name)
		if base == "" || base == "." || base == ".." {
			return effectiveTarget{}, fmt.Errorf("destination must name a regular file")
		}
		if dir == "" {
			dir = "."
		}
		parent, err := fs.EvalSymlinks(dir)
		if err != nil {
			return effectiveTarget{}, fmt.Errorf("failed to resolve destination parent: %w", err)
		}
		resolved = filepath.Join(parent, base)
		info = nil
	}
	return effectiveTarget{path: resolved, info: info}, nil
}
