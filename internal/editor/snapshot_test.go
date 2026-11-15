package editor

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/config"
	"golang.org/x/tools/txtar"
)

// TXTAR cases drive the real edit/validate/save workflow using temporary files
// and an editor callback. Comment-only fstab inputs require no live observations.
func TestStaleEditFixtures(t *testing.T) {
	paths, err := filepath.Glob("testdata/stale_edit/*.txtar")
	if err != nil || len(paths) == 0 {
		t.Fatalf("fixtures: %v", err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			files := map[string][]byte{}
			for _, f := range txtar.Parse(data).Files {
				files[f.Name] = f.Data
			}
			root := t.TempDir()
			dest := filepath.Join(root, "fstab")
			backup := filepath.Join(root, "backup")
			original, present := files["original"]
			if present {
				if err := os.WriteFile(dest, original, 0640); err != nil {
					t.Fatal(err)
				}
			}
			action := strings.TrimSpace(string(files["action"]))
			if action == "retarget" {
				if err := os.Rename(dest, filepath.Join(root, "first")); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "second"), files["external"], 0640); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("first", dest); err != nil {
					t.Fatal(err)
				}
			}
			sm := &mockSystemdManager{isActive: true}
			editorCalls := 0
			deps := Dependencies{FS: osFS{}, Systemd: sm, Editor: func(path string) error {
				editorCalls++
				got, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(got, original) {
					t.Fatalf("initial editor content %q (%v), want %q", got, err, original)
				}
				switch action {
				case "content", "appear":
					if err := os.WriteFile(dest, files["external"], 0640); err != nil {
						t.Fatal(err)
					}
				case "same-stat-content":
					before, err := os.Stat(dest)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(dest, files["external"], 0640); err != nil {
						t.Fatal(err)
					}
					if err := os.Chtimes(dest, before.ModTime(), before.ModTime()); err != nil {
						t.Fatal(err)
					}
				case "replace":
					// Allocate the new inode before unlinking the old one, avoiding inode reuse.
					replacement := filepath.Join(root, "replacement")
					if err := os.WriteFile(replacement, original, 0640); err != nil {
						t.Fatal(err)
					}
					if err := os.Rename(replacement, dest); err != nil {
						t.Fatal(err)
					}
				case "delete":
					if err := os.Remove(dest); err != nil {
						t.Fatal(err)
					}
				case "retarget":
					if err := os.Remove(dest); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink("second", dest); err != nil {
						t.Fatal(err)
					}
				case "unchanged":
				default:
					t.Fatalf("unknown fixture action %q", action)
				}
				return os.WriteFile(path, files["edited"], 0600)
			}}
			cfg := config.Config{FilePath: dest, BackupDir: backup, SystemdReload: config.SystemdReloadAuto}
			err = EditAndLint(cfg, &mockAPI{}, deps)
			if editorCalls != 1 {
				t.Fatalf("editor calls %d", editorCalls)
			}
			if action == "unchanged" {
				if err != nil {
					t.Fatal(err)
				}
				got, err := os.ReadFile(dest)
				if err != nil || !bytes.Equal(got, files["edited"]) {
					t.Fatalf("save = %q %v", got, err)
				}
				if !hasRCSArchive(backup) {
					t.Fatalf("missing successful revision under %s", backup)
				}
				return
			}
			if !errors.Is(err, ErrConflict) {
				t.Fatalf("want conflict, got %v", err)
			}
			var recovery *RecoveryError
			if !errors.As(err, &recovery) {
				t.Fatalf("missing recovery path: %v", err)
			}
			t.Cleanup(func() { _ = os.Remove(recovery.Path) })
			got, readErr := os.ReadFile(recovery.Path)
			if readErr != nil || !bytes.Equal(got, files["edited"]) {
				t.Fatalf("recovery = %q %v", got, readErr)
			}
			info, statErr := os.Stat(recovery.Path)
			if statErr != nil || info.Mode().Perm() != 0600 {
				t.Fatalf("recovery mode = %v %v", info, statErr)
			}
			if _, err := os.Stat(backup); !os.IsNotExist(err) {
				t.Fatalf("conflict created backup history: %v", err)
			}
			if sm.reloadCount != 0 {
				t.Fatalf("reload after conflict: %d", sm.reloadCount)
			}
			if action == "delete" {
				if _, err := os.Stat(dest); !os.IsNotExist(err) {
					t.Fatalf("deleted destination recreated: %v", err)
				}
			} else {
				want := files["external"]
				if action == "replace" {
					want = original
				}
				got, err := os.ReadFile(dest)
				if err != nil || !bytes.Equal(got, want) {
					t.Fatalf("external content lost: %q %v, want %q", got, err, want)
				}
			}
			if action == "retarget" {
				link, err := os.Readlink(dest)
				if err != nil || link != "second" {
					t.Fatalf("link changed %q %v", link, err)
				}
				got, err := os.ReadFile(filepath.Join(root, "first"))
				if err != nil || !bytes.Equal(got, original) {
					t.Fatalf("old target changed %q %v", got, err)
				}
			}
		})
	}
}

type snapshotReadFS struct {
	FS
	readErr   error
	afterRead func()
}

func (f *snapshotReadFS) ReadFile(name string) ([]byte, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	data, err := f.FS.ReadFile(name)
	if f.afterRead != nil {
		f.afterRead()
	}
	return data, err
}

func TestSnapshotReadFailureDoesNotLaunchEditor(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "fstab")
	if err := os.WriteFile(dest, []byte("# original\n"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	err := EditAndLint(config.Config{FilePath: dest}, &mockAPI{}, Dependencies{FS: &snapshotReadFS{FS: osFS{}, readErr: os.ErrPermission}, Editor: func(string) error { called = true; return nil }})
	if !errors.Is(err, os.ErrPermission) || called {
		t.Fatalf("snapshot failure = %v, editor=%v", err, called)
	}
}

func TestSnapshotChangeWhileReadingDoesNotLaunchEditor(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "fstab")
	replacement := filepath.Join(root, "replacement")
	for _, p := range []string{dest, replacement} {
		if err := os.WriteFile(p, []byte("# same\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	called := false
	fs := &snapshotReadFS{FS: osFS{}, afterRead: func() {
		if err := os.Rename(replacement, dest); err != nil {
			t.Fatal(err)
		}
	}}
	err := EditAndLint(config.Config{FilePath: dest}, &mockAPI{}, Dependencies{FS: fs, Editor: func(string) error { called = true; return nil }})
	if !errors.Is(err, ErrConflict) || called {
		t.Fatalf("inconsistent snapshot = %v, editor=%v", err, called)
	}
}

func TestSnapshotUnavailableAtCommitPreservesEdits(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "fstab")
	if err := os.WriteFile(dest, []byte("# original\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fs := &snapshotReadFS{FS: osFS{}}
	edited := []byte("# edited\n")
	err := EditAndLint(config.Config{FilePath: dest}, &mockAPI{}, Dependencies{FS: fs, Editor: func(path string) error {
		fs.readErr = os.ErrPermission
		return os.WriteFile(path, edited, 0600)
	}})
	if !errors.Is(err, os.ErrPermission) || errors.Is(err, ErrConflict) {
		t.Fatalf("unknown metadata misreported: %v", err)
	}
	var recovery *RecoveryError
	if !errors.As(err, &recovery) {
		t.Fatalf("no recovery: %v", err)
	}
	defer func() { _ = os.Remove(recovery.Path) }()
	got, readErr := os.ReadFile(recovery.Path)
	if readErr != nil || !bytes.Equal(got, edited) {
		t.Fatalf("recovery = %q %v", got, readErr)
	}
	got, readErr = os.ReadFile(dest)
	if readErr != nil || string(got) != "# original\n" {
		t.Fatalf("destination changed: %q %v", got, readErr)
	}
}

func TestSnapshotRecheckedAfterDestinationPreparation(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "fstab")
	if err := os.WriteFile(dest, []byte("# original\n"), 0600); err != nil {
		t.Fatal(err)
	}
	original, _, err := snapshotDestination(dest, osFS{})
	if err != nil {
		t.Fatal(err)
	}
	fs := &targetFaultFS{FS: osFS{}, beforeClose: func() {
		if err := os.WriteFile(dest, []byte("# external\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}}
	res := saveAndVersion(config.Config{FilePath: dest}, []byte("# edited\n"), fs, &original)
	if !errors.Is(res.Err, ErrConflict) || res.Saved || fs.renameCalled {
		t.Fatalf("late change overwritten: %+v rename=%v", res, fs.renameCalled)
	}
}

func TestConflictDoesNotTouchExistingHistoryOrReload(t *testing.T) {
	fs := newCommitTestFS()
	sm := &mockSystemdManager{isActive: true}
	history := []byte("untouched history")
	if err := fs.WriteFile("/backup/fstab,v", history, 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{FilePath: "/etc/fstab", BackupDir: "/backup", SystemdReload: config.SystemdReloadAuto}
	err := EditAndLint(cfg, &mockAPI{}, Dependencies{FS: fs, Systemd: sm, Editor: func(path string) error {
		if err := fs.WriteFile(cfg.FilePath, []byte("# external\n"), 0600); err != nil {
			t.Fatal(err)
		}
		return os.WriteFile(path, []byte("# edited\n"), 0600)
	}})
	if !errors.Is(err, ErrConflict) || sm.reloadCount != 0 {
		t.Fatalf("conflict result %v reload=%d", err, sm.reloadCount)
	}
	var recovery *RecoveryError
	if !errors.As(err, &recovery) {
		t.Fatalf("no recovery: %v", err)
	}
	got, readErr := fs.ReadFile("/backup/fstab,v")
	if readErr != nil || !bytes.Equal(got, history) {
		t.Fatalf("history changed: %q %v", got, readErr)
	}
}

type recoveryFailFS struct{ FS }

func (f recoveryFailFS) CreateTemp(dir, pattern string) (TempFile, error) {
	if pattern == "vifstab-recovery-*" {
		return nil, os.ErrPermission
	}
	return f.FS.CreateTemp(dir, pattern)
}
func TestRecoveryFailureRetainsEditorCopy(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "fstab")
	if err := os.WriteFile(dest, []byte("# original\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var editedPath string
	err := EditAndLint(config.Config{FilePath: dest}, &mockAPI{}, Dependencies{FS: recoveryFailFS{FS: osFS{}}, Editor: func(path string) error {
		editedPath = path
		if err := os.WriteFile(dest, []byte("# external\n"), 0600); err != nil {
			t.Fatal(err)
		}
		return os.WriteFile(path, []byte("# edited\n"), 0600)
	}})
	defer func() { _ = os.Remove(editedPath) }()
	if !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), editedPath) || !strings.Contains(err.Error(), "retained") {
		t.Fatalf("unrecoverable diagnostic: %v", err)
	}
	got, readErr := os.ReadFile(editedPath)
	if readErr != nil || string(got) != "# edited\n" {
		t.Fatalf("editor copy discarded: %q %v", got, readErr)
	}
}

func TestRecoveryWriteFailuresRemoveIncompleteCopies(t *testing.T) {
	for _, op := range []string{"temp.create", "temp.write", "temp.chmod", "temp.sync", "temp.close"} {
		t.Run(op, func(t *testing.T) {
			fs := &mockFS{mfs: fstest.MapFS{}, faults: map[string]error{"destination." + op: os.ErrPermission}}
			path, err := preserveRecovery([]byte("private edited content"), fs)
			if !errors.Is(err, os.ErrPermission) || path != "" {
				t.Fatalf("incomplete recovery accepted: %q, %v", path, err)
			}
			if len(fs.mfs) != 0 {
				t.Fatalf("incomplete recovery files retained: %v", fs.mfs)
			}
		})
	}
}

func hasRCSArchive(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ",v") {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}
