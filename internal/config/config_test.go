package config

import (
	"testing"
)

func TestGetDefaultConfig(t *testing.T) {
	tests := []struct {
		name          string
		xdgCacheHome  string
		homeEnv       string
		userProfile   string // for windows
		filePath      string
		expectedCache string
	}{
		{
			name:          "XDG_CACHE_HOME is set",
			xdgCacheHome:  "/custom/xdg/cache",
			homeEnv:       "/home/user",
			filePath:      "/etc/fstab",
			expectedCache: "/custom/xdg/cache/vifstab",
		},
		{
			name:          "XDG_CACHE_HOME is empty, HOME is set",
			xdgCacheHome:  "",
			homeEnv:       "/home/user",
			filePath:      "/etc/fstab",
			expectedCache: "/home/user/.cache/vifstab",
		},
		{
			name:          "Neither XDG_CACHE_HOME nor HOME is set",
			xdgCacheHome:  "",
			homeEnv:       "",
			userProfile:   "",
			filePath:      "/etc/fstab",
			expectedCache: "/tmp/.vifstab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CACHE_HOME", tt.xdgCacheHome)
			t.Setenv("HOME", tt.homeEnv)
			t.Setenv("USERPROFILE", tt.userProfile)

			cfg := GetDefaultConfig(tt.filePath)
			if cfg.BackupDir != tt.expectedCache {
				t.Errorf("expected BackupDir %q, got %q", tt.expectedCache, cfg.BackupDir)
			}
			if cfg.FilePath != tt.filePath {
				t.Errorf("expected FilePath %q, got %q", tt.filePath, cfg.FilePath)
			}
		})
	}
}
