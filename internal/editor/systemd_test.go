package editor

import (
	"bytes"
	"io"
	"os"
	"testing"
	"errors"

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
func captureOutput(f func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := f()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String(), err
}

func TestHandleSystemdReload(t *testing.T) {
	tests := []struct {
		name              string
		filePath          string
		isActive          bool
		hadBlockingIssues bool
		systemdReload     config.SystemdReloadAction
		reloadErr         error
		expectedOut       string
		expectReload      bool
		expectErr         bool
	}{
		{
			name:              "valid /etc/fstab + active systemd + auto -> Reload called once",
			filePath:          "/etc/fstab",
			isActive:          true,
			hadBlockingIssues: false,
			systemdReload:     config.SystemdReloadAuto,
			expectedOut:       "note: systemd daemon-reload completed.",
			expectReload:      true,
			expectErr:         false,
		},
		{
			name:              "valid /etc/fstab + active systemd + warn -> diagnostic, no Reload",
			filePath:          "/etc/fstab",
			isActive:          true,
			hadBlockingIssues: false,
			systemdReload:     config.SystemdReloadWarn,
			expectedOut:       "warning: systemd mount-unit state predates this fstab change",
			expectReload:      false,
			expectErr:         false,
		},
		{
			name:              "valid /etc/fstab + active systemd + never -> no Reload",
			filePath:          "/etc/fstab",
			isActive:          true,
			hadBlockingIssues: false,
			systemdReload:     config.SystemdReloadNever,
			expectedOut:       "",
			expectReload:      false,
			expectErr:         false,
		},
		{
			name:              "non-/etc/fstab -> no systemd interaction",
			filePath:          "/etc/fstab.old",
			isActive:          true,
			hadBlockingIssues: false,
			systemdReload:     config.SystemdReloadAuto,
			expectedOut:       "",
			expectReload:      false,
			expectErr:         false,
		},
		{
			name:              "inactive systemd -> no Reload",
			filePath:          "/etc/fstab",
			isActive:          false,
			hadBlockingIssues: false,
			systemdReload:     config.SystemdReloadAuto,
			expectedOut:       "",
			expectReload:      false,
			expectErr:         false,
		},
		{
			name:              "BLOCKING VALIDATION + continue -> Reload MUST NOT be called",
			filePath:          "/etc/fstab",
			isActive:          true,
			hadBlockingIssues: true,
			systemdReload:     config.SystemdReloadAuto,
			expectedOut:       "",
			expectReload:      false,
			expectErr:         false,
		},
		{
			name:              "active systemd + auto + failure -> error returned",
			filePath:          "/etc/fstab",
			isActive:          true,
			hadBlockingIssues: false,
			systemdReload:     config.SystemdReloadAuto,
			reloadErr:         errors.New("mock reload failed"),
			expectedOut:       "",
			expectReload:      true,
			expectErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				FilePath:      tt.filePath,
				SystemdReload: tt.systemdReload,
			}
			sm := &mockSystemdManager{
				isActive:  tt.isActive,
				reloadErr: tt.reloadErr,
			}

			out, err := captureOutput(func() error {
				return HandleSystemdReload(cfg, sm, tt.hadBlockingIssues)
			})

			if tt.expectReload != sm.reloaded {
				t.Errorf("expected reload=%v, got %v", tt.expectReload, sm.reloaded)
			}
			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectedOut != "" && !bytes.Contains([]byte(out), []byte(tt.expectedOut)) {
				t.Errorf("expected output containing %q, got %q", tt.expectedOut, out)
			}
		})
	}
}
