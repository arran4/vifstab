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
		{"continued after blocking validation", "/etc/fstab", config.SystemdReloadAuto, true, false, 0, "warning: systemd", false},
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

func TestCommitEditRCSFailureAfterSaveDoesNotReload(t *testing.T) {
	fsLayer := &rcsFailFS{mockFS: newCommitTestFS()}
	sm := &mockSystemdManager{isActive: true}
	cfg := config.Config{FilePath: "/etc/fstab", BackupDir: "/backup", SystemdReload: config.SystemdReloadAuto}
	err := commitEdit(cfg, []byte("new"), true, Dependencies{FS: fsLayer, Systemd: sm})
	if err == nil || !strings.Contains(err.Error(), "rcs update failed") {
		t.Fatalf("got error %v, want rcs failure", err)
	}
	if sm.reloadCount != 0 {
		t.Fatalf("Reload called %d times after rcs failure, want 0", sm.reloadCount)
	}
}

type rcsFailFS struct{ *mockFS }

func (f *rcsFailFS) CreateTemp(dir, pattern string) (TempFile, error) {
	if strings.Contains(dir, "/backup") {
		return nil, errors.New("rcs create temp failed")
	}
	return f.mockFS.CreateTemp(dir, pattern)
}

func (f *rcsFailFS) Rename(oldpath, newpath string) error {
	if strings.Contains(newpath, "/backup") {
		return errors.New("rcs rename failed")
	}
	return f.mockFS.Rename(oldpath, newpath)
}

func (f *rcsFailFS) WriteFile(name string, data []byte, mode os.FileMode) error {
	if strings.Contains(name, "/backup") {
		return errors.New("rcs write failed")
	}
	return f.mockFS.WriteFile(name, data, mode)
}

func TestCommitEditSaveFailureBeforeRCSDoesNotReload(t *testing.T) {
	fsLayer := &failingWriteFS{mockFS: newCommitTestFS()}
	sm := &mockSystemdManager{isActive: true}
	cfg := config.Config{FilePath: "/etc/fstab", BackupDir: "/backup", SystemdReload: config.SystemdReloadAuto}
	err := commitEdit(cfg, []byte("new"), true, Dependencies{FS: fsLayer, Systemd: sm})
	if err == nil || !strings.Contains(err.Error(), "create temp failed") {
		t.Fatalf("got error %v, want destination failure", err)
	}
	if sm.reloadCount != 0 {
		t.Fatalf("Reload called %d times after failed save", sm.reloadCount)
	}
}

func TestCommitEditDestinationFailureBeforeRCSSuccessDoesNotReload(t *testing.T) {
	fsLayer := &destinationFailFS{mockFS: newCommitTestFS()}
	sm := &mockSystemdManager{isActive: true}
	cfg := config.Config{FilePath: "/etc/fstab", BackupDir: "/backup", SystemdReload: config.SystemdReloadAuto}
	err := commitEdit(cfg, []byte("new"), true, Dependencies{FS: fsLayer, Systemd: sm})
	if err == nil || !strings.Contains(err.Error(), "destination create temp failed") {
		t.Fatalf("got error %v, want destination failure", err)
	}
	if _, rcsErr := fsLayer.ReadFile("/backup/fstab,v"); !os.IsNotExist(rcsErr) {
		t.Fatalf("RCS file should not exist, got err: %v", rcsErr)
	}
	if sm.reloadCount != 0 {
		t.Fatalf("Reload called %d times after destination failure", sm.reloadCount)
	}
}

type failingWriteFS struct{ *mockFS }

func (f *failingWriteFS) CreateTemp(dir, pattern string) (TempFile, error) {
	return nil, errors.New("create temp failed")
}

func (f *failingWriteFS) WriteFile(string, []byte, os.FileMode) error {
	return errors.New("write failed")
}

type destinationFailFS struct{ *mockFS }

func (f *destinationFailFS) CreateTemp(dir, pattern string) (TempFile, error) {
	if dir == "/etc" {
		return nil, errors.New("destination create temp failed")
	}
	return f.mockFS.CreateTemp(dir, pattern)
}

func (f *destinationFailFS) Rename(oldpath, newpath string) error {
	if newpath == "/etc/fstab" {
		return errors.New("destination rename failed")
	}
	return f.mockFS.Rename(oldpath, newpath)
}

func (f *destinationFailFS) WriteFile(name string, data []byte, mode os.FileMode) error {
	if name == "/etc/fstab" {
		return errors.New("destination write failed")
	}
	return f.mockFS.WriteFile(name, data, mode)
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

func TestCommitEditRCSFailureSuppressesReloadFailure(t *testing.T) {
	fsLayer := newCommitTestFS()
	rcsErr := errors.New("rcs failure")
	fsLayer.faults = map[string]error{
		"rcs.mkdir": rcsErr,
	}
	systemdErr := errors.New("systemd failure")
	sm := &mockSystemdManager{isActive: true, reloadErr: systemdErr}
	cfg := config.Config{FilePath: "/etc/fstab", BackupDir: "/backup", SystemdReload: config.SystemdReloadAuto}

	err := commitEdit(cfg, []byte("new"), true, Dependencies{FS: fsLayer, Systemd: sm})
	if err == nil {
		t.Fatal("expected RCS error, got nil")
	}

	if !errors.Is(err, rcsErr) {
		t.Errorf("error does not wrap RCS failure: %v", err)
	}

	if errors.Is(err, systemdErr) || sm.reloadCount != 0 {
		t.Errorf("reload must not run after RCS failure: calls=%d, error=%v", sm.reloadCount, err)
	}
}

type blockingWorkflowAPI struct{ mockAPI }

func (blockingWorkflowAPI) CheckFileExists(string) rules.FileExistsResult {
	return rules.FileDoesNotExist
}

func (blockingWorkflowAPI) CheckBlockDeviceExists(string) rules.ExistState { return rules.DoesNotExist }

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
	out, editErr := captureOutput(t, func() error { return EditAndLint(cfg, &blockingWorkflowAPI{}, deps) })
	if editErr != nil {
		t.Fatalf("EditAndLint: %v", editErr)
	}
	if sm.reloadCount != 0 {
		t.Fatalf("Reload called %d times after blocking validation", sm.reloadCount)
	}
	if !strings.Contains(out, "warning: systemd") {
		t.Fatalf("missing stale-systemd warning after overridden validation: %q", out)
	}
	got, err := fsLayer.ReadFile("/etc/fstab")
	if err != nil || !bytes.Contains(got, []byte("/dev/missing")) {
		t.Fatalf("explicit save did not commit edited file: %q, %v", got, err)
	}
}

func TestSaveWithoutValidationUsesConservativeSystemdWarning(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		action      config.SystemdReloadAction
		active      bool
		wantWarning bool
	}{
		{"auto", "/etc/fstab", config.SystemdReloadAuto, true, true},
		{"warn", "/etc/fstab", config.SystemdReloadWarn, true, true},
		{"never", "/etc/fstab", config.SystemdReloadNever, true, false},
		{"inactive systemd", "/etc/fstab", config.SystemdReloadWarn, false, false},
		{"non-/etc/fstab", "/etc/fstab.test", config.SystemdReloadWarn, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsLayer := newCommitTestFS()
			sm := &mockSystemdManager{isActive: tt.active}
			cfg := config.Config{FilePath: tt.filePath, SystemdReload: tt.action}
			out, err := captureOutput(t, func() error {
				return SaveWithoutValidation(cfg, []byte("formatted"), Dependencies{FS: fsLayer, Systemd: sm})
			})
			if err != nil {
				t.Fatal(err)
			}
			if sm.reloadCount != 0 {
				t.Fatalf("Reload called %d times for unvalidated formatted content", sm.reloadCount)
			}
			hasWarning := strings.Contains(out, "warning: systemd")
			if hasWarning != tt.wantWarning {
				t.Fatalf("warning present = %v, want %v; output: %q", hasWarning, tt.wantWarning, out)
			}
			if tt.wantWarning && !strings.Contains(out, "currently mounted filesystems may also require a remount or reboot") {
				t.Fatalf("missing mounted-filesystem note: %q", out)
			}
			got, readErr := fsLayer.ReadFile(tt.filePath)
			if readErr != nil || string(got) != "formatted" {
				t.Fatalf("formatted content was not saved: %q, %v", got, readErr)
			}
		})
	}
}

// A rename may succeed while its durability sync fails. Preserve the installed
// content, but do not automatically reload after an operation returning failure.
func TestCommitEditDirectorySyncFailureDoesNotReload(t *testing.T) {
	fsLayer := &syncFailFS{mockFS: newCommitTestFS()}
	sm := &mockSystemdManager{isActive: true}
	cfg := config.Config{FilePath: "/etc/fstab", SystemdReload: config.SystemdReloadAuto}
	err := commitEdit(cfg, []byte("new"), true, Dependencies{FS: fsLayer, Systemd: sm})
	if err == nil || !strings.Contains(err.Error(), "sync directory") {
		t.Fatalf("expected durability error: %v", err)
	}
	if sm.reloadCount != 0 {
		t.Fatalf("Reload called %d times after sync failure", sm.reloadCount)
	}
	got, readErr := fsLayer.ReadFile("/etc/fstab")
	if readErr != nil || string(got) != "new" {
		t.Fatalf("installed content: %q, %v", got, readErr)
	}
}

type syncFailFS struct{ *mockFS }

func (f *syncFailFS) SyncDir(string) error { return errors.New("injected directory sync failure") }
