package rcs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	rcslib "github.com/arran4/golang-rcs"
	"golang.org/x/tools/txtar"
)

// Source paths are identities only here: no real /etc/fstab or source device is
// opened. All history artifacts are created under the test's temporary directory.
func TestHistoryTXTAR(t *testing.T) {
	paths, err := filepath.Glob("testdata/history/*.txtar")
	if err != nil || len(paths) == 0 {
		t.Fatalf("fixtures: %v", err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var cases []struct {
				Source    string   `json:"source"`
				Revisions []string `json:"revisions_base64"`
			}
			archive := txtar.Parse(data)
			if len(archive.Files) != 1 {
				t.Fatal("expected one cases.json section")
			}
			if err := json.Unmarshal(archive.Files[0].Data, &cases); err != nil {
				t.Fatal(err)
			}
			cfg := Config{Dir: t.TempDir()}
			seen := map[string]bool{}
			for _, tc := range cases {
				history, err := HistoryPath(tc.Source, cfg.Dir, nil)
				if err != nil {
					t.Fatal(err)
				}
				if seen[history] {
					t.Fatalf("source identities collided: %s", history)
				}
				seen[history] = true
				rel, err := filepath.Rel(cfg.Dir, history)
				if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
					t.Fatalf("history escaped backup dir: %s %v", history, err)
				}
				if len(filepath.Base(history)) > 110 {
					t.Fatalf("unbounded basename: %d", len(filepath.Base(history)))
				}
				for _, encoded := range tc.Revisions {
					content, err := base64.StdEncoding.DecodeString(encoded)
					if err != nil {
						t.Fatal(err)
					}
					if err := Update(tc.Source, content, cfg); err != nil {
						t.Fatalf("update %q: %v", tc.Source, err)
					}
				}
				for i, encoded := range tc.Revisions {
					want, err := base64.StdEncoding.DecodeString(encoded)
					if err != nil {
						t.Fatal(err)
					}
					revision := fmt.Sprintf("1.%d", i+1)
					got, err := Restore(tc.Source, revision, cfg, nil)
					if err != nil || !bytes.Equal(got, want) {
						t.Fatalf("restore %s = %q (%v), want %q", revision, got, err, want)
					}
				}
				first, err := HistoryPath(tc.Source, cfg.Dir, nil)
				if err != nil || first != history {
					t.Fatalf("mapping changed: %s %v", first, err)
				}
				info, err := os.Stat(history)
				if err != nil || info.Mode().Perm() != 0600 {
					t.Fatalf("archive mode: %v %v", info, err)
				}
			}
		})
	}
}

type cwdFS struct {
	FS
	wd  string
	err error
}

func (f cwdFS) Getwd() (string, error) { return f.wd, f.err }
func TestSourcePathIdentity(t *testing.T) {
	cfg := t.TempDir()
	a, err := HistoryPath("fstab", cfg, cwdFS{FS: osFS{}, wd: "/one"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := HistoryPath("/one/fstab", cfg, nil)
	if err != nil || a != b {
		t.Fatalf("relative identity: %s %s %v", a, b, err)
	}
	c, err := HistoryPath("fstab", cfg, cwdFS{FS: osFS{}, wd: "/two"})
	if err != nil || a == c {
		t.Fatalf("working directories collided: %s %s %v", a, c, err)
	}
	d, err := HistoryPath("link/../fstab", cfg, cwdFS{FS: osFS{}, wd: "/one"})
	if err != nil || a == d {
		t.Fatalf("symlink-sensitive spelling collapsed: %s %s %v", a, d, err)
	}
	if _, err := HistoryPath("fstab", cfg, cwdFS{FS: osFS{}, err: os.ErrPermission}); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("working directory unknown: %v", err)
	}
	for _, source := range []string{"", "bad\x00path"} {
		if _, err := HistoryPath(source, cfg, nil); err == nil {
			t.Fatalf("invalid source accepted %q", source)
		}
	}
	if _, err := HistoryPath("fstab", "", nil); err == nil {
		t.Fatal("empty backup directory accepted")
	}
}

func TestLegacyHistoryRequiresExplicitReconciliation(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}
	legacy := filepath.Join(cfg.Dir, "fstab,v")
	old := []byte("unlabelled legacy history, potentially mixed\n")
	if err := os.WriteFile(legacy, old, 0600); err != nil {
		t.Fatal(err)
	}
	sources := []string{"/etc/fstab", "/srv/chroot/etc/fstab"}
	for _, source := range sources {
		err := Update(source, []byte(source), cfg)
		var conflict *LegacyHistoryError
		if !errors.As(err, &conflict) || conflict.LegacyPath != legacy {
			t.Fatalf("legacy adopted silently: %v", err)
		}
		got, readErr := os.ReadFile(legacy)
		if readErr != nil || !bytes.Equal(got, old) {
			t.Fatalf("legacy changed: %q %v", got, readErr)
		}
		history, err := HistoryPath(source, cfg.Dir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(history); !os.IsNotExist(err) {
			t.Fatalf("new history created despite legacy uncertainty: %v", err)
		}
	}
	// Model the documented owner action: retain the original elsewhere, then start
	// independent histories. The product never renames/imports the legacy archive.
	retained := filepath.Join(cfg.Dir, "legacy-retained,v")
	if err := os.Rename(legacy, retained); err != nil {
		t.Fatal(err)
	}
	for _, source := range sources {
		if err := Update(source, []byte(source), cfg); err != nil {
			t.Fatal(err)
		}
		got, err := Restore(source, "1.1", cfg, nil)
		if err != nil || string(got) != source {
			t.Fatalf("new history mixed: %q %v", got, err)
		}
	}
	got, err := os.ReadFile(retained)
	if err != nil || !bytes.Equal(got, old) {
		t.Fatalf("retained legacy changed: %q %v", got, err)
	}
}

func TestConcurrentHistoryUpdatesKeepEveryRevision(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}
	source := "/fixtures/fstab"
	const count = 24
	var wg sync.WaitGroup
	failures := make(chan error, count)
	for i := range count {
		wg.Add(1)
		go func(i int) { defer wg.Done(); failures <- Update(source, []byte(fmt.Sprintf("# writer %d\n", i)), cfg) }(i)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	seen := map[string]bool{}
	for i := range count {
		got, err := Restore(source, fmt.Sprintf("1.%d", i+1), cfg, nil)
		if err != nil {
			t.Fatal(err)
		}
		if seen[string(got)] {
			t.Fatalf("duplicate/lost content %q", got)
		}
		seen[string(got)] = true
	}
	for i := range count {
		if !seen[fmt.Sprintf("# writer %d\n", i)] {
			t.Fatalf("writer %d was lost", i)
		}
	}
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

type orderedFS struct {
	FS
	held              bool
	checks            int
	lockErr, closeErr error
}

func (f *orderedFS) Lock(name string) (io.Closer, error) {
	if f.lockErr != nil {
		return nil, f.lockErr
	}
	lock, err := f.FS.Lock(name)
	if err != nil {
		return nil, err
	}
	f.held = true
	return closerFunc(func() error { f.held = false; return errors.Join(lock.Close(), f.closeErr) }), nil
}
func (f *orderedFS) check() error {
	if !f.held {
		return errors.New("history access outside lock")
	}
	f.checks++
	return nil
}
func (f *orderedFS) Lstat(name string) (os.FileInfo, error) {
	if err := f.check(); err != nil {
		return nil, err
	}
	return f.FS.Lstat(name)
}
func (f *orderedFS) ReadFile(name string) ([]byte, error) {
	if err := f.check(); err != nil {
		return nil, err
	}
	return f.FS.ReadFile(name)
}
func (f *orderedFS) CreateTemp(dir, pattern string) (TempFile, error) {
	if err := f.check(); err != nil {
		return nil, err
	}
	return f.FS.CreateTemp(dir, pattern)
}
func (f *orderedFS) Rename(a, b string) error {
	if err := f.check(); err != nil {
		return err
	}
	return f.FS.Rename(a, b)
}
func (f *orderedFS) SyncDir(name string) error {
	if err := f.check(); err != nil {
		return err
	}
	return f.FS.SyncDir(name)
}
func TestHistoryLockCoversReadModifyWrite(t *testing.T) {
	fs := &orderedFS{FS: osFS{}}
	cfg := Config{Dir: t.TempDir()}
	for _, s := range []string{"first", "second"} {
		if err := Update("/fixture/fstab", []byte(s), cfg, fs); err != nil {
			t.Fatal(err)
		}
	}
	if fs.held || fs.checks < 10 {
		t.Fatalf("incomplete lock scope: held=%v checks=%d", fs.held, fs.checks)
	}
	fs.lockErr = os.ErrPermission
	if err := Update("/fixture/fstab", []byte("not committed"), cfg, fs); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("lock failure ignored %v", err)
	}
	fs.lockErr = nil
	fs.closeErr = os.ErrPermission
	if err := Update("/fixture/fstab", []byte("committed, unlock error"), cfg, fs); !errors.Is(err, os.ErrPermission) || fs.held {
		t.Fatalf("unlock failure ignored %v", err)
	}
}

func TestCorruptOrForeignHistoryIsNeverOverwritten(t *testing.T) {
	for _, kind := range []string{"source", "checksum", "delta", "unreachable"} {
		t.Run(kind, func(t *testing.T) {
			cfg := Config{Dir: t.TempDir()}
			source := "/fixture/fstab"
			for _, s := range []string{"old\n", "new\n"} {
				if err := Update(source, []byte(s), cfg); err != nil {
					t.Fatal(err)
				}
			}
			path, err := HistoryPath(source, cfg.Dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			f, err := rcslib.ParseFile(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "source":
				f.Description = strings.ReplaceAll(f.Description, base64.StdEncoding.EncodeToString([]byte(source)), base64.StdEncoding.EncodeToString([]byte("/foreign/fstab")))
			case "checksum":
				f.RevisionContents[0].Text = "tampered"
			case "delta":
				f.RevisionContents[1].Text = "a0 999999999\ntruncated\n"
			case "unreachable":
				f.RevisionHeads[0].NextRevision = ""
			}
			damaged := []byte(f.String())
			if err := os.WriteFile(path, damaged, 0600); err != nil {
				t.Fatal(err)
			}
			if err := Update(source, []byte("would overwrite"), cfg); err == nil {
				t.Fatal("damaged history accepted")
			}
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, damaged) {
				t.Fatalf("damaged history overwritten %v", err)
			}
			if _, err := Restore(source, "1.1", cfg, nil); err == nil {
				t.Fatal("unverifiable recovery returned")
			}
		})
	}
}
func TestHistorySymlinksAreRefused(t *testing.T) {
	for _, suffix := range []string{"", ".lock"} {
		t.Run(suffix, func(t *testing.T) {
			cfg := Config{Dir: t.TempDir()}
			source := "/fixture/fstab"
			history, err := HistoryPath(source, cfg.Dir, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(history), 0700); err != nil {
				t.Fatal(err)
			}
			sentinel := filepath.Join(cfg.Dir, "sentinel")
			if err := os.WriteFile(sentinel, []byte("untouched"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(sentinel, history+suffix); err != nil {
				t.Fatal(err)
			}
			if err := Update(source, []byte("edited"), cfg); err == nil {
				t.Fatal("symlink accepted")
			}
			got, err := os.ReadFile(sentinel)
			if err != nil || string(got) != "untouched" {
				t.Fatalf("symlink target changed %q %v", got, err)
			}
		})
	}
}
func TestBackupDirectorySymlinkDotDotIsNotCleaned(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data", "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("data/sub", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Dir: root + "/link/../backup"}
	source := "/fixture/fstab"
	if err := Update(source, []byte("correct directory"), cfg); err != nil {
		t.Fatal(err)
	}
	path, err := HistoryPath(source, filepath.Join(root, "data", "backup"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("backup path meaning changed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "backup")); !os.IsNotExist(err) {
		t.Fatalf("lexically cleaned backup created: %v", err)
	}
}
func TestReplacementDeltaBounds(t *testing.T) {
	for _, s := range []string{"d1 0\n", "d1 -1\n", "a0 999999999\nx\n", "a0 1\nunterminated", "d2 1\n", "d1 1\nd1 1\n", "a0 1\nx\ntrailing"} {
		if _, err := parseReplacementDelta(s); err == nil {
			t.Errorf("malformed delta accepted %q", s)
		}
	}
}

func TestRootWorkingDirectoryIdentity(t *testing.T) {
	a, err := HistoryPath("fstab", "/backup", cwdFS{FS: osFS{}, wd: "/"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := HistoryPath("/fstab", "/backup", nil)
	if err != nil || a != b {
		t.Fatalf("root relative path changed identity: %s %s %v", a, b, err)
	}
}
func TestInvalidAuthorCannotBreakArchive(t *testing.T) {
	t.Setenv("USER", "bad;author\nhead 9.9;")
	cfg := Config{Dir: t.TempDir()}
	if err := Update("/fixture/fstab", []byte("safe\n"), cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Restore("/fixture/fstab", "", cfg, nil)
	if err != nil || string(got) != "safe\n" {
		t.Fatalf("author affected content %q %v", got, err)
	}
}
