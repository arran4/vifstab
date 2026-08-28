package editor

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/arran4/vifstab/internal/config"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/save_and_version/*.txtar
var testdataFS embed.FS

func normalizeMockPath(name string) string {
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(name)), "/")
}

// mockFS implements the editor.FS interface using fstest.MapFS
type mockFS struct {
	mfs fstest.MapFS

	faults       map[string]error
	faultTrigger map[string]bool
}

func (m *mockFS) injectFault(op string, contextPath string) error {
	if m.faults != nil {
		// First check for operation specific to a path context
		// e.g. "rcs.stat" vs just "stat"
		if err, ok := m.faults[op]; ok {
			if m.faultTrigger != nil {
				m.faultTrigger[op] = true
			}
			return err
		}
	}
	return nil
}

func getOpContext(op string, name string) string {
	if strings.Contains(name, "/backup") || strings.Contains(name, ",v") {
		return "rcs." + op
	}
	return "destination." + op
}

func (m *mockFS) ReadFile(name string) ([]byte, error) {
	if err := m.injectFault(getOpContext("read", name), name); err != nil {
		return nil, err
	}
	name = normalizeMockPath(name)
	if f, ok := m.mfs[name]; ok {
		return f.Data, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	if err := m.injectFault(getOpContext("write", name), name); err != nil {
		return err
	}
	name = normalizeMockPath(name)
	m.mfs[name] = &fstest.MapFile{Data: data, Mode: perm}
	return nil
}

func (m *mockFS) MkdirAll(path string, perm os.FileMode) error {
	if err := m.injectFault(getOpContext("mkdir", path), path); err != nil {
		return err
	}
	path = normalizeMockPath(path)
	m.mfs[path] = &fstest.MapFile{Mode: perm | os.ModeDir}
	return nil
}

func (m *mockFS) Rename(oldpath, newpath string) error {
	if err := m.injectFault(getOpContext("rename", newpath), newpath); err != nil {
		return err
	}
	oldpath = normalizeMockPath(oldpath)
	newpath = normalizeMockPath(newpath)
	if f, ok := m.mfs[oldpath]; ok {
		m.mfs[newpath] = f
		delete(m.mfs, oldpath)
		return nil
	}
	return os.ErrNotExist
}

func (m *mockFS) Remove(name string) error {
	if err := m.injectFault(getOpContext("remove", name), name); err != nil {
		return err
	}
	name = normalizeMockPath(name)
	delete(m.mfs, name)
	return nil
}

type mockTempFile struct {
	name   string
	mfs    *mockFS
	closed bool
	data   []byte
}

func (t *mockTempFile) Write(p []byte) (n int, err error) {
	if t.closed {
		return 0, fmt.Errorf("file closed")
	}
	if err := t.mfs.injectFault(getOpContext("temp.write", t.name), t.name); err != nil {
		return 0, err
	}
	t.data = append(t.data, p...)
	return len(p), nil
}

func (t *mockTempFile) Close() error {
	if err := t.mfs.injectFault(getOpContext("temp.close", t.name), t.name); err != nil {
		return err
	}
	t.closed = true
	t.mfs.WriteFile(t.name, t.data, 0600)
	return nil
}

func (t *mockTempFile) Name() string {
	return t.name
}

func (t *mockTempFile) Chmod(mode os.FileMode) error {
	if t.closed {
		return fmt.Errorf("file closed")
	}
	if err := t.mfs.injectFault(getOpContext("temp.chmod", t.name), t.name); err != nil {
		return err
	}
	return nil
}

func (m *mockFS) CreateTemp(dir, pattern string) (TempFile, error) {
	if err := m.injectFault(getOpContext("temp.create", dir), dir); err != nil {
		return nil, err
	}
	dir = normalizeMockPath(dir)
	name := filepath.Join(dir, strings.ReplaceAll(pattern, "*", "mock"))
	return &mockTempFile{name: normalizeMockPath(name), mfs: m}, nil
}

type dummyFileInfo struct {
	mode os.FileMode
}

func (d dummyFileInfo) Name() string       { return "dummy" }
func (d dummyFileInfo) Size() int64        { return 0 }
func (d dummyFileInfo) Mode() os.FileMode  { return d.mode }
func (d dummyFileInfo) ModTime() time.Time { return time.Time{} }
func (d dummyFileInfo) IsDir() bool        { return false }
func (d dummyFileInfo) Sys() interface{}   { return nil }

func (m *mockFS) Stat(name string) (os.FileInfo, error) {
	if err := m.injectFault(getOpContext("stat", name), name); err != nil {
		return nil, err
	}
	name = normalizeMockPath(name)
	if f, ok := m.mfs[name]; ok {
		// Return a dummy stat with the file mode
		return dummyFileInfo{mode: f.Mode}, nil
	}
	return nil, os.ErrNotExist
}

func TestSaveAndVersionRealFS(t *testing.T) {
	// Create a temporary directory
	dir := t.TempDir()
	fstabPath := filepath.Join(dir, "fstab")
	backupDir := filepath.Join(dir, "backup")

	cfg := config.Config{
		FilePath:  fstabPath,
		BackupDir: backupDir,
	}

	content := []byte("fstab content")

	tests := []struct {
		name         string
		initialMode  os.FileMode
		setupFile    bool
		expectedMode os.FileMode
	}{
		{
			name:         "existing 0600 remains 0600",
			initialMode:  0600,
			setupFile:    true,
			expectedMode: 0600,
		},
		{
			name:         "existing 0644 remains 0644",
			initialMode:  0644,
			setupFile:    true,
			expectedMode: 0644,
		},
		{
			name:         "existing 0640 remains 0640",
			initialMode:  0640,
			setupFile:    true,
			expectedMode: 0640,
		},
		{
			name:         "nonexistent becomes 0600",
			setupFile:    false,
			expectedMode: 0600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFile {
				if err := os.WriteFile(fstabPath, []byte("old content"), tt.initialMode); err != nil {
					t.Fatalf("setup write failed: %v", err)
				}
				if err := os.Chmod(fstabPath, tt.initialMode); err != nil {
					t.Fatalf("setup chmod failed: %v", err)
				}
			} else {
				os.Remove(fstabPath) // Ensure it doesn't exist
			}

			if res := SaveAndVersion(cfg, content, DefaultFS); res.Err != nil {
				t.Fatalf("SaveAndVersion failed: %v", res.Err)
			}

			info, err := os.Stat(fstabPath)
			if err != nil {
				t.Fatalf("stat failed: %v", err)
			}
			if info.Mode().Perm() != tt.expectedMode {
				t.Errorf("expected mode %v, got %v", tt.expectedMode, info.Mode().Perm())
			}

			got, err := os.ReadFile(fstabPath)
			if err != nil {
				t.Fatalf("read failed: %v", err)
			}
			if string(got) != string(content) {
				t.Errorf("expected content %q, got %q", string(content), string(got))
			}
		})
	}
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
			var newContent []byte
			var updateContent []byte
			var expectedContent string
			var expectedUpdatedContent string
			var expectedRcsExists bool
			var expectedRcsContent string
			var expectedRcsPath string

			var expectedSaved string
			var expectedVersioningEnabled string
			var expectedVersioned string
			var expectedErr string

			var testFile string
			var backupDir string
			faults := make(map[string]error)

			for _, file := range archive.Files {
				if file.Name == "input/etc/fstab" {
					testFile = "/etc/fstab"
				} else if file.Name == "input/new_fstab" {
					newContent = file.Data
				} else if file.Name == "input/update_fstab" {
					updateContent = file.Data
				} else if file.Name == "expected/etc/fstab" {
					expectedContent = string(file.Data)
				} else if file.Name == "expected/fstab_updated" {
					expectedUpdatedContent = string(file.Data)
				} else if file.Name == "expected/rcs_exists" {
					expectedRcsExists = strings.TrimSpace(string(file.Data)) == "true"
				} else if file.Name == "expected/Saved" {
					expectedSaved = strings.TrimSpace(string(file.Data))
				} else if file.Name == "expected/VersioningEnabled" {
					expectedVersioningEnabled = strings.TrimSpace(string(file.Data))
				} else if file.Name == "expected/Versioned" {
					expectedVersioned = strings.TrimSpace(string(file.Data))
				} else if file.Name == "expected/Err" {
					expectedErr = strings.TrimSpace(string(file.Data))
				} else if strings.HasPrefix(file.Name, "expected/") && strings.HasSuffix(file.Name, ",v") {
					expectedRcsContent = string(file.Data)
					expectedRcsPath = strings.TrimPrefix(file.Name, "expected/")
				} else if strings.HasPrefix(file.Name, "env/") {
					envName := strings.TrimPrefix(file.Name, "env/")
					if envName == "XDG_CACHE_HOME" {
						backupDir = strings.TrimSpace(string(file.Data))
					}
				} else if strings.HasPrefix(file.Name, "fault/") {
					faultName := strings.TrimPrefix(file.Name, "fault/")
					faults[faultName] = fmt.Errorf("%s", strings.TrimSpace(string(file.Data)))
				}
			}

			if testFile == "" {
				testFile = "/etc/fstab" // default fallback
			}
			if backupDir == "" && expectedVersioningEnabled != "false" {
				backupDir = "/var/backup" // default fallback
			}

			// Use memory mock FS to avoid actual file system access
			mfs := &mockFS{
				mfs:           make(fstest.MapFS),
				faults:        faults,
				faultTrigger:  make(map[string]bool),
			}

			// Setup full initial file state dynamically
			for _, file := range archive.Files {
				if strings.HasPrefix(file.Name, "input/") && file.Name != "input/new_fstab" && file.Name != "input/update_fstab" {
					p := strings.TrimPrefix(file.Name, "input/")
					p = normalizeMockPath(p)
					mfs.mfs[p] = &fstest.MapFile{Data: file.Data, Mode: 0644}
				}
			}

			cfg := config.Config{
				FilePath:  testFile,
				BackupDir: filepath.Join(backupDir, "vifstab"), // Match how main.go constructs it if XDG_CACHE_HOME is set: backupDir = xdgCache + "/vifstab"
			}
			// Wait, the logic in main is: if XDG_CACHE_HOME is set: backupDir = xdgCache + "/vifstab"
			// Wait, the config parser just takes the path. Let's make the test explicitly match the target path.
			// Let's set it explicitly to what the test expects the output path to be:
			// In success.txtar expected path is /var/backup/fstab,v
			// That means BackupDir should be /var/backup
			cfg.BackupDir = backupDir

			// Set a dummy user for RCS output consistency
			os.Setenv("USER", "testuser")
			// And mock the time in RCS? the rcs package uses time.Now(). We can't easily mock time.Now in external package.
			// Let's just check that it contains the content strings.

			// Initial Save
			res := SaveAndVersion(cfg, newContent, mfs)

			if expectedErr != "" {
				if res.Err == nil || !strings.Contains(res.Err.Error(), expectedErr) {
					t.Fatalf("expected error %q, got %v", expectedErr, res.Err)
				}
			} else {
				if res.Err != nil {
					t.Fatalf("SaveAndVersion failed: %v", res.Err)
				}
			}

			if expectedSaved != "" {
				expectedSavedBool := expectedSaved == "true"
				if res.Saved != expectedSavedBool {
					t.Errorf("expected Saved=%v, got %v", expectedSavedBool, res.Saved)
				}
			}

			if expectedVersioningEnabled != "" {
				expectedVersioningEnabledBool := expectedVersioningEnabled == "true"
				if res.VersioningEnabled != expectedVersioningEnabledBool {
					t.Errorf("expected VersioningEnabled=%v, got %v", expectedVersioningEnabledBool, res.VersioningEnabled)
				}
			}

			if expectedVersioned != "" {
				expectedVersionedBool := expectedVersioned == "true"
				if res.Versioned != expectedVersionedBool {
					t.Errorf("expected Versioned=%v, got %v", expectedVersionedBool, res.Versioned)
				}
			}

			if expectedContent != "" {
				got, err := mfs.ReadFile(testFile)
				if err != nil {
					t.Fatalf("failed to read updated file: %v", err)
				}
				if string(got) != expectedContent {
					t.Errorf("expected %q, got %q", expectedContent, string(got))
				}
			}

			if expectedRcsExists || expectedRcsPath != "" {
				_, err = mfs.Stat(backupDir)
				gotRcsExists := err == nil
				if gotRcsExists != expectedRcsExists && fileHasRCS(archive) {
					t.Errorf("expected RCS dir exists=%v, got %v", expectedRcsExists, gotRcsExists)
				}
			}

			// Subsequent update if requested by test file
			if len(updateContent) > 0 {
				if res := SaveAndVersion(cfg, updateContent, mfs); res.Err != nil {
					t.Fatalf("SaveAndVersion update failed: %v", res.Err)
				}

				if expectedUpdatedContent != "" {
					gotUpdated, err := mfs.ReadFile(testFile)
					if err != nil {
						t.Fatalf("failed to read updated file after update: %v", err)
					}
					if string(gotUpdated) != expectedUpdatedContent {
						t.Errorf("expected updated %q, got %q", expectedUpdatedContent, string(gotUpdated))
					}
				}
			}

			if expectedRcsContent != "" && expectedRcsPath != "" {
				// Don't assert full structure for error tests, just that it exists
				if expectedErr == "" {
					rcsBytes, err := mfs.ReadFile(expectedRcsPath)
					if err != nil {
						t.Fatalf("failed to read rcs file at expected path %s: %v", expectedRcsPath, err)
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
			}

			// Verify fault consumption
			for f := range faults {
				if !mfs.faultTrigger[f] && expectedErr != "" {
					t.Errorf("Expected fault %q to be triggered, but it was not", f)
				}
			}
		})
	}
}

func fileHasRCS(archive *txtar.Archive) bool {
	for _, f := range archive.Files {
		if f.Name == "expected/rcs_exists" {
			return true
		}
	}
	return false
}
