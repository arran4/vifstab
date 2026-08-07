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
			var updateContent []byte
			var expectedContent string
			var expectedUpdatedContent string
			var expectedRcsExists bool
			var expectedRcsContent string

			// Support optional configurations from outside "input" and "expected"
			var testFile = "/etc/fstab"
			var backupDir = "/var/backup"

			for _, file := range archive.Files {
				if file.Name == "input/initial_fstab" {
					initialContent = file.Data
				} else if file.Name == "input/new_fstab" {
					newContent = file.Data
				} else if file.Name == "input/update_fstab" {
					updateContent = file.Data
				} else if file.Name == "expected/fstab" {
					expectedContent = string(file.Data)
				} else if file.Name == "expected/fstab_updated" {
					expectedUpdatedContent = string(file.Data)
				} else if file.Name == "expected/rcs_exists" {
					expectedRcsExists = strings.TrimSpace(string(file.Data)) == "true"
				} else if file.Name == "expected/rcs_content" {
					expectedRcsContent = string(file.Data)
				} else if file.Name == "config/filepath" {
					testFile = strings.TrimSpace(string(file.Data))
				} else if file.Name == "config/backupdir" {
					backupDir = strings.TrimSpace(string(file.Data))
				}
			}

			// Use memory mock FS to avoid actual file system access
			mfs := &mockFS{mfs: make(fstest.MapFS)}

			if len(initialContent) > 0 {
				mfs.WriteFile(testFile, initialContent, 0644)
			}

			cfg := config.Config{
				FilePath:  testFile,
				BackupDir: backupDir,
			}

			// Set a dummy user for RCS output consistency
			os.Setenv("USER", "testuser")
			// And mock the time in RCS? the rcs package uses time.Now(). We can't easily mock time.Now in external package.
			// Let's just check that it contains the content strings.

			// Initial Save
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

			// Subsequent update if requested by test file
			if len(updateContent) > 0 {
				if err := SaveAndVersion(cfg, updateContent, mfs); err != nil {
					t.Fatalf("SaveAndVersion update failed: %v", err)
				}

				gotUpdated, err := mfs.ReadFile(testFile)
				if err != nil {
					t.Fatalf("failed to read updated file after update: %v", err)
				}
				if string(gotUpdated) != expectedUpdatedContent {
					t.Errorf("expected updated %q, got %q", expectedUpdatedContent, string(gotUpdated))
				}
			}

			if expectedRcsContent != "" {
				rcsFilepath := filepath.Join(backupDir, filepath.Base(testFile)+",v")
				rcsBytes, err := mfs.ReadFile(rcsFilepath)
				if err != nil {
					t.Fatalf("failed to read rcs file: %v", err)
				}

				// Time and exact formatting might differ, so let's verify key parts
				rcsStr := string(rcsBytes)

				if !strings.Contains(rcsStr, "desc") {
					t.Errorf("expected rcs file to contain 'desc', got:\n%s", rcsStr)
				}

				if len(updateContent) > 0 {
					if !strings.Contains(rcsStr, "1.2") {
						t.Errorf("expected rcs file to contain '1.2' revision, got:\n%s", rcsStr)
					}
					if !strings.Contains(rcsStr, "@even newer fstab content\n@") {
						t.Errorf("expected rcs file to contain '@even newer fstab content\\n@', got:\n%s", rcsStr)
					}
				} else {
					if !strings.Contains(rcsStr, "1.1") {
						t.Errorf("expected rcs file to contain '1.1' revision, got:\n%s", rcsStr)
					}
					if !strings.Contains(rcsStr, "@new fstab content\n@") {
						t.Errorf("expected rcs file to contain '@new fstab content\\n@', got:\n%s", rcsStr)
					}
				}
			}
		})
	}
}
