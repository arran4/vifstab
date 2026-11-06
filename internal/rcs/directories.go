package rcs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MakeDurableDirs persists newly created directory entries, including nested
// backup ancestors. Existing ancestors need no additional read permission.
// The complete operation remains injectable as FS.MkdirAll.
func MakeDurableDirs(path string, mode os.FileMode) error {
	return makeDurableDirs(path, mode, os.Stat, os.MkdirAll, (osFS{}).SyncDir)
}

func directoryParent(path string) string {
	path = strings.TrimRight(path, string(filepath.Separator))
	index := strings.LastIndexByte(path, filepath.Separator)
	if index < 0 {
		return "."
	}
	if index == 0 {
		return string(filepath.Separator)
	}
	return path[:index]
}

func makeDurableDirs(path string, mode os.FileMode, stat func(string) (os.FileInfo, error), mkdir func(string, os.FileMode) error, syncDir func(string) error) error {
	var missing []string
	for current := path; ; current = directoryParent(current) {
		info, err := stat(current)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("RCS backup parent is not a directory")
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		missing = append(missing, current)
		if directoryParent(current) == current {
			return fmt.Errorf("cannot resolve RCS backup ancestor")
		}
	}
	if err := mkdir(path, mode); err != nil {
		return err
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if err := syncDir(directoryParent(missing[i])); err != nil {
			return fmt.Errorf("failed to sync RCS directory creation: %w", err)
		}
	}
	return nil
}
