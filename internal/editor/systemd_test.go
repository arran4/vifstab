package editor

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/rules"
)

type mockSystemdManager struct {
	isActive    bool
	reloadErr   error
	reloadCount int
	onReload    func()
}

func (m *mockSystemdManager) IsActive() bool { return m.isActive }
func (m *mockSystemdManager) Reload() error {
	m.reloadCount++
	if m.onReload != nil {
		m.onReload()
	}
	return m.reloadErr
}

func captureOutput(t *testing.T, f func() error) (string, error) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	callErr := f()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String(), callErr
}

func newCommitTestFS() *mockFS {
	return &mockFS{mfs: fstest.MapFS{
		"etc/fstab": &fstest.MapFile{Data: []byte("old"), Mode: 0644},
	}}
}

func TestCommitEditSystemdHandling(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		action           config.SystemdReloadAction
		active           bool
		validationPassed bool
		wantReloads      int
		wantOutput       string
		wantNoSystemdOut bool
	}{
		{"auto", "/etc/fstab", config.SystemdReloadAuto, true, true, 1, "daemon-reload completed", false},
		{"warn diagnostic", "/etc/fstab", config.SystemdReloadWarn, true, true, 0, "warning: systemd", false},
		{"never", "/etc/fstab", config.SystemdReloadNever, true, true, 0, "", true},
		{"other file", "/etc/fstab.test", config.SystemdReloadAuto, true, true, 0, "", true},
		{"inactive", "/etc/fstab", config.SystemdReloadAuto, false, true, 0, "", true},
		{"continued after blocking validation", "/etc/fstab", config.SystemdReloadAuto, true, false, 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsLayer := newCommitTestFS()
			sm := &mockSystemdManager{isActive: tt.active}
			cfg := config.Config{FilePath: tt.path, SystemdReload: tt.action}
			out, err := captureOutput(t, func() error {
				return commitEdit(cfg, []byte("new"), tt.validationPassed, Dependencies{FS: fsLayer, Systemd: sm})
			})
			if err != nil {
				t.Fatalf("commitEdit: %v", err)
			}
			if sm.reloadCount != tt.wantReloads {
				t.Fatalf("Reload called %d times, want %d", sm.reloadCount, tt.wantReloads)
			}
			if tt.wantOutput != "" && !strings.Contains(out, tt.wantOutput) {
				t.Fatalf("output %q does not contain %q", out, tt.wantOutput)
			}
			if tt.wantNoSystemdOut && strings.Contains(out, "systemd") {
				t.Fatalf("unexpected systemd output: %q", out)
			}
			if got, readErr := fsLayer.ReadFile(tt.path); readErr != nil || string(got) != "new" {
				t.Fatalf("saved content = %q, %v", got, readErr)
			}
		})
	}
}

func TestCommitEditSaveFailureDoesNotReload(t *testing.T) {
	fsLayer := &failingWriteFS{mockFS: newCommitTestFS()}
	sm := &mockSystemdManager{isActive: true}
	cfg := config.Config{FilePath: "/etc/fstab", BackupDir: "/backup", SystemdReload: config.SystemdReloadAuto}
	err := commitEdit(cfg, []byte("new"), true, Dependencies{FS: fsLayer, Systemd: sm})
	if err == nil || !strings.Contains(err.Error(), "rcs update failed") {
		t.Fatalf("got error %v, want RCS failure", err)
	}
	if sm.reloadCount != 0 {
		t.Fatalf("Reload called %d times after failed save", sm.reloadCount)
	}
}

type failingWriteFS struct{ *mockFS }

func (f *failingWriteFS) WriteFile(string, []byte, os.FileMode) error {
	return errors.New("write failed")
}

func TestCommitEditSavesBeforeSystemdHandling(t *testing.T) {
	fsLayer := newCommitTestFS()
	sm := &mockSystemdManager{isActive: true}
	sm.onReload = func() {
		got, err := fsLayer.ReadFile("/etc/fstab")
		if err != nil || string(got) != "new" {
			t.Fatalf("Reload occurred before committed save: %q, %v", got, err)
		}
	}
	cfg := config.Config{FilePath: "/etc/fstab", SystemdReload: config.SystemdReloadAuto}
	if err := commitEdit(cfg, []byte("new"), true, Dependencies{FS: fsLayer, Systemd: sm}); err != nil {
		t.Fatal(err)
	}
}

func TestCommitEditReloadFailureIsPostSaveError(t *testing.T) {
	fsLayer := newCommitTestFS()
	reloadErr := errors.New("access denied")
	sm := &mockSystemdManager{isActive: true, reloadErr: reloadErr}
	cfg := config.Config{FilePath: "/etc/fstab", SystemdReload: config.SystemdReloadAuto}
	err := commitEdit(cfg, []byte("new"), true, Dependencies{FS: fsLayer, Systemd: sm})
	if err == nil || !strings.Contains(err.Error(), "fstab saved, but systemd daemon-reload failed: access denied") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !errors.Is(err, reloadErr) {
		t.Fatalf("error does not wrap reload failure: %v", err)
	}
	got, readErr := fsLayer.ReadFile("/etc/fstab")
	if readErr != nil || string(got) != "new" {
		t.Fatalf("saved fstab was rolled back: %q, %v", got, readErr)
	}
}

type blockingWorkflowAPI struct{ mockAPI }

func (blockingWorkflowAPI) CheckFileExists(string) rules.FileExistsResult {
	return rules.FileDoesNotExist
}

func (blockingWorkflowAPI) CheckBlockDeviceExists(string) bool { return false }

func TestEditAndLintContinueAfterBlockingValidationDoesNotReload(t *testing.T) {
	fsLayer := newCommitTestFS()
	sm := &mockSystemdManager{isActive: true}
	deps := Dependencies{
		FS:      fsLayer,
		Systemd: sm,
		Input:   strings.NewReader("c\n"),
		Editor: func(path string) error {
			return os.WriteFile(path, []byte("/dev/missing /definitely/missing ext4 defaults 0\n"), 0600)
		},
	}
	cfg := config.Config{FilePath: "/etc/fstab", SystemdReload: config.SystemdReloadAuto}
	if err := EditAndLint(cfg, &blockingWorkflowAPI{}, deps); err != nil {
		t.Fatalf("EditAndLint: %v", err)
	}
	if sm.reloadCount != 0 {
		t.Fatalf("Reload called %d times after blocking validation", sm.reloadCount)
	}
	got, err := fsLayer.ReadFile("/etc/fstab")
	if err != nil || !bytes.Contains(got, []byte("/dev/missing")) {
		t.Fatalf("explicit save did not commit edited file: %q, %v", got, err)
	}
}
