package editor

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/config"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/save_and_version/*.txtar
var testdataFS embed.FS

// mockFS implements the editor.FS interface using fstest.MapFS
type mockFS struct {
	mfs fstest.MapFS
}

func (m *mockFS) ReadFile(name string) ([]byte, error) {
	name = strings.TrimPrefix(filepath.ToSlash(name), "/")
	if f, ok := m.mfs[name]; ok {
		return f.Data, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	name = strings.TrimPrefix(filepath.ToSlash(name), "/")
	m.mfs[name] = &fstest.MapFile{Data: data, Mode: perm}
	return nil
}

func (m *mockFS) MkdirAll(path string, perm os.FileMode) error {
	path = strings.TrimPrefix(filepath.ToSlash(path), "/")
	m.mfs[path] = &fstest.MapFile{Mode: perm | os.ModeDir}
	return nil
}

func (m *mockFS) Rename(oldpath, newpath string) error {
	oldpath = strings.TrimPrefix(filepath.ToSlash(oldpath), "/")
	newpath = strings.TrimPrefix(filepath.ToSlash(newpath), "/")
	if f, ok := m.mfs[oldpath]; ok {
		m.mfs[newpath] = f
		delete(m.mfs, oldpath)
		return nil
	}
	return os.ErrNotExist
}

func (m *mockFS) Stat(name string) (os.FileInfo, error) {
	name = strings.TrimPrefix(filepath.ToSlash(name), "/")
	if _, ok := m.mfs[name]; ok {
		// Just return a dummy non-error stat
		return nil, nil
	}
	return nil, os.ErrNotExist
}


func TestSaveAndVersion(t *testing.T) {
	entries, err := fs.ReadDir(testdataFS, "testdata/save_and_version")
	if err != nil {
		t.Fatalf("failed to read testdata dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txtar") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			b, err := fs.ReadFile(testdataFS, "testdata/save_and_version/"+entry.Name())
			if err != nil {
				t.Fatalf("failed to read test file: %v", err)
			}

			archive := txtar.Parse(b)
			var initialContent []byte
			var newContent []byte
			var expectedContent string
			var expectedRcsExists bool

			for _, file := range archive.Files {
				if file.Name == "input/initial_fstab" {
					initialContent = file.Data
				} else if file.Name == "input/new_fstab" {
					newContent = file.Data
				} else if file.Name == "expected/fstab" {
					expectedContent = string(file.Data)
				} else if file.Name == "expected/rcs_exists" {
					expectedRcsExists = strings.TrimSpace(string(file.Data)) == "true"
				}
			}

			// Use memory mock FS to avoid actual file system access
			mfs := &mockFS{mfs: make(fstest.MapFS)}

			testFile := "/etc/fstab"
			backupDir := "/var/backup"

			if len(initialContent) > 0 {
				mfs.WriteFile(testFile, initialContent, 0644)
			}

			cfg := config.Config{
				FilePath:  testFile,
				BackupDir: backupDir,
			}

			if err := SaveAndVersion(cfg, newContent, mfs); err != nil {
				t.Fatalf("SaveAndVersion failed: %v", err)
			}

			got, err := mfs.ReadFile(testFile)
			if err != nil {
				t.Fatalf("failed to read updated file: %v", err)
			}
			if string(got) != expectedContent {
				t.Errorf("expected %q, got %q", expectedContent, string(got))
			}

			_, err = mfs.Stat(backupDir)
			gotRcsExists := err == nil
			if gotRcsExists != expectedRcsExists {
				t.Errorf("expected RCS dir exists=%v, got %v", expectedRcsExists, gotRcsExists)
			}
		})
	}
}
