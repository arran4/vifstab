package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetDefaultConfig(filePath string) string {
	backupDir := ""
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		backupDir = xdgCache + "/vifstab"
	} else if home, err := os.UserHomeDir(); err == nil {
		backupDir = home + "/.cache/vifstab"
	} else {
		backupDir, _ = os.MkdirTemp("", "vifstab-*")
	}

	return backupDir
}

func main() {
	os.Setenv("XDG_CACHE_HOME", "")
	os.Setenv("HOME", "")
	os.Setenv("USERPROFILE", "")
	fmt.Println(GetDefaultConfig("/etc/fstab"))
}
