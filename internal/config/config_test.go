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
		wantErr       bool
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
			name:          "Alternative requires no cache",
			filePath:      "/tmp/example.fstab",
			expectedCache: "",
		},
		{
			name:          "Neither XDG_CACHE_HOME nor HOME is set",
			xdgCacheHome:  "",
			homeEnv:       "",
			userProfile:   "",
			filePath:      "/etc/fstab",
			expectedCache: "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CACHE_HOME", tt.xdgCacheHome)
			t.Setenv("HOME", tt.homeEnv)
			t.Setenv("USERPROFILE", tt.userProfile)

			cfg, err := GetDefaultConfig(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if cfg.FilePath != tt.filePath {
				t.Errorf("expected FilePath %q, got %q", tt.filePath, cfg.FilePath)
			}
			if tt.wantErr {
				return
			}
			if cfg.BackupDir != tt.expectedCache {
				t.Errorf("expected BackupDir %q, got %q", tt.expectedCache, cfg.BackupDir)
			}
		})
	}
}
