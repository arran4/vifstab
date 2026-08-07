package config

import "os"

type Config struct {
	BackupDir string
	FilePath  string
}

func GetDefaultConfig(filePath string) Config {
	backupDir := ""
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		backupDir = xdgCache + "/vifstab"
	} else if home, err := os.UserHomeDir(); err == nil {
		backupDir = home + "/.cache/vifstab"
	} else {
		backupDir = "/tmp/.vifstab"
	}

	return Config{
		BackupDir: backupDir,
		FilePath:  filePath,
	}
}
