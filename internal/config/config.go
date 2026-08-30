package config

import "os"

type SystemdReloadAction string

const (
	SystemdReloadWarn  SystemdReloadAction = "warn"
	SystemdReloadAuto  SystemdReloadAction = "auto"
	SystemdReloadNever SystemdReloadAction = "never"
)

type Config struct {
	BackupDir     string
	FilePath      string
	SystemdReload SystemdReloadAction
}

func GetDefaultConfig(filePath string) Config {
	backupDir := ""
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		backupDir = xdgCache + "/vifstab"
	} else if home, err := os.UserHomeDir(); err == nil {
		backupDir = home + "/.cache/vifstab"
	} else {
		backupDir, _ = os.MkdirTemp("", "vifstab-*")
	}

	return Config{
		BackupDir:     backupDir,
		FilePath:      filePath,
		SystemdReload: SystemdReloadWarn,
	}
}
