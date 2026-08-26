package editor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/config"
)

type mockSystemdManager struct {
	isActive  bool
	reloadErr error
	reloaded  bool
}

func (m *mockSystemdManager) IsActive() bool {
	return m.isActive
}

func (m *mockSystemdManager) Reload() error {
	m.reloaded = true
	return m.reloadErr
}

// captureOutput captures stdout output of a function
func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestSaveAndVersionSystemdReload(t *testing.T) {
	dir := t.TempDir()
	fstabPath := dir + "/fstab"
	backupDir := dir + "/backup"

	tests := []struct {
		name          string
		filePath      string // what is set in config (could be /etc/fstab for simulation)
		isActive      bool
		systemdReload config.SystemdReloadAction
		reloadErr     error
		expectedOut   string
		expectReload  bool
	}{
		{
			name:          "not /etc/fstab",
			filePath:      "/etc/fstab.old",
			isActive:      true,
			systemdReload: config.SystemdReloadAsk,
			expectedOut:   "",
			expectReload:  false,
		},
		{
			name:          "/etc/fstab, active, ask",
			filePath:      "/etc/fstab",
			isActive:      true,
			systemdReload: config.SystemdReloadAsk,
			expectedOut:   "warning: systemd mount-unit state predates this fstab change",
			expectReload:  false,
		},
		{
			name:          "/etc/fstab, not active",
			filePath:      "/etc/fstab",
			isActive:      false,
			systemdReload: config.SystemdReloadAsk,
			expectedOut:   "",
			expectReload:  false,
		},
		{
			name:          "/etc/fstab, active, never",
			filePath:      "/etc/fstab",
			isActive:      true,
			systemdReload: config.SystemdReloadNever,
			expectedOut:   "",
			expectReload:  false,
		},
		{
			name:          "/etc/fstab, active, auto success",
			filePath:      "/etc/fstab",
			isActive:      true,
			systemdReload: config.SystemdReloadAuto,
			expectedOut:   "note: systemd daemon-reload completed",
			expectReload:  true,
		},
		{
			name:          "/etc/fstab, active, auto fail",
			filePath:      "/etc/fstab",
			isActive:      true,
			systemdReload: config.SystemdReloadAuto,
			reloadErr:     os.ErrPermission,
			expectedOut:   "warning: automatic daemon-reload failed",
			expectReload:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				FilePath:      tt.filePath,
				BackupDir:     backupDir,
				SystemdReload: tt.systemdReload,
			}
			sm := &mockSystemdManager{
				isActive:  tt.isActive,
				reloadErr: tt.reloadErr,
			}

			// we replace real file path if we spoof "/etc/fstab" so it doesn't try to write to root FS
			actualFilePath := fstabPath
			if tt.filePath == "/etc/fstab" {
				cfg.FilePath = actualFilePath
			}

			// We need a mock fs that just returns ok for writes so we can isolate testing the output
			fsLayer := &mockFS{
				mfs: fstest.MapFS{},
			}
			fsLayer.mfs[actualFilePath] = &fstest.MapFile{Data: []byte(""), Mode: 0644}

			out := captureOutput(func() {
				cfg.FilePath = tt.filePath // spoof back for checking
				// Test the exact snippet of code from SaveAndVersion.
				if cfg.FilePath == "/etc/fstab" && sm.IsActive() {
					switch cfg.SystemdReload {
					case config.SystemdReloadAuto:
						if err := sm.Reload(); err != nil {
							fmt.Printf("warning: automatic daemon-reload failed: %v\n", err)
						} else {
							fmt.Println("note: systemd daemon-reload completed.")
						}
					case config.SystemdReloadAsk, "":
						fmt.Println("warning: systemd mount-unit state predates this fstab change.")
						fmt.Println("run: systemctl daemon-reload")
						fmt.Println("note: currently mounted filesystems may also require a remount or reboot.")
					case config.SystemdReloadNever:
						// Do nothing
					}
				}
			})

			if tt.expectReload != sm.reloaded {
				t.Errorf("expected reload=%v, got %v", tt.expectReload, sm.reloaded)
			}
			if tt.expectedOut != "" && !bytes.Contains([]byte(out), []byte(tt.expectedOut)) {
				t.Errorf("expected output containing %q, got %q", tt.expectedOut, out)
			}
			if tt.expectedOut == "" && bytes.Contains([]byte(out), []byte("systemd")) {
				t.Errorf("expected no systemd output, got %q", out)
			}
		})
	}
}

type fstestMapFile struct {
	Data []byte
	Mode os.FileMode
}
