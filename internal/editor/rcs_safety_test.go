package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/rcs"
)

// Exercise the production editor->RCS adapter too: its general source-file
// reader permits symlinked fstab paths, but history reads must remain no-follow.
func TestSaveRefusesSymlinkHistoryThroughAdapter(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{FilePath: filepath.Join(root, "fstab"), BackupDir: filepath.Join(root, "backup")}
	history, err := rcs.HistoryPath(cfg.FilePath, cfg.BackupDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(history), 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(root, "sentinel")
	if err := os.WriteFile(sentinel, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sentinel, history); err != nil {
		t.Fatal(err)
	}
	res := SaveAndVersion(cfg, []byte("new target\n"), osFS{})
	if res.Err == nil || !res.Saved || res.Versioned {
		t.Fatalf("misreported partial save: %+v", res)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "untouched" {
		t.Fatalf("foreign history target changed: %q %v", got, err)
	}
	link, err := os.Readlink(history)
	if err != nil || link != sentinel {
		t.Fatalf("history symlink replaced: %q %v", link, err)
	}
}

type historySwapFS struct {
	FS
	history, target string
	swapped         bool
}

func (f *historySwapFS) Lstat(name string) (os.FileInfo, error) {
	info, err := f.FS.Lstat(name)
	if err == nil && name == f.history && !f.swapped {
		if err := os.Remove(name); err != nil {
			return nil, err
		}
		if err := os.Symlink(f.target, name); err != nil {
			return nil, err
		}
		f.swapped = true
	}
	return info, err
}
func TestSaveDetectsHistorySymlinkBeforeRead(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{FilePath: filepath.Join(root, "fstab"), BackupDir: filepath.Join(root, "backup")}
	if err := rcs.Update(cfg.FilePath, []byte("# original\n"), rcs.Config{Dir: cfg.BackupDir}); err != nil {
		t.Fatal(err)
	}
	history, err := rcs.HistoryPath(cfg.FilePath, cfg.BackupDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(history)
	if err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(root, "foreign-history")
	if err := os.WriteFile(foreign, original, 0600); err != nil {
		t.Fatal(err)
	}
	fs := &historySwapFS{FS: osFS{}, history: history, target: foreign}
	res := SaveAndVersion(cfg, []byte("# edited\n"), fs)
	if res.Err == nil || !res.Saved || res.Versioned {
		t.Fatalf("history substitution accepted: %+v", res)
	}
	link, err := os.Readlink(history)
	if err != nil || link != foreign {
		t.Fatalf("substituted link replaced: %q %v", link, err)
	}
	got, err := os.ReadFile(foreign)
	if err != nil || string(got) != string(original) {
		t.Fatalf("foreign history changed: %v", err)
	}
}
