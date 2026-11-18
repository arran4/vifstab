package config

import (
	"fmt"
	"os"
)

type SystemdReloadAction string

const (
	SystemdReloadWarn  SystemdReloadAction = "warn"
	SystemdReloadAuto  SystemdReloadAction = "auto"
	SystemdReloadNever SystemdReloadAction = "never"
)

type MetadataPolicy string

const (
	MetadataPolicyStrict     MetadataPolicy = "strict"
	MetadataPolicySupported  MetadataPolicy = "supported"
	MetadataPolicyBestEffort MetadataPolicy = "best-effort"
)

type Config struct {
	BackupDir      string
	FilePath       string
	SystemdReload  SystemdReloadAction
	MetadataPolicy MetadataPolicy
}

func GetDefaultConfig(filePath string) (Config, error) {
	backupDir := ""
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		backupDir = xdgCache + "/vifstab"
	} else if userCache, err := os.UserCacheDir(); err == nil {
		backupDir = userCache + "/vifstab"
	} else {
		return Config{
			FilePath:       filePath,
			SystemdReload:  SystemdReloadWarn,
			MetadataPolicy: MetadataPolicyStrict,
		}, fmt.Errorf("could not determine user cache directory: %w. Please specify an explicit backup directory via --backup-dir", err)
	}

	return Config{
		BackupDir:      backupDir,
		FilePath:       filePath,
		SystemdReload:  SystemdReloadWarn,
		MetadataPolicy: MetadataPolicyStrict,
	}, nil
}
