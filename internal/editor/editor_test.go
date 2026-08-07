package editor

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/config"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/save_and_version/*.txtar
var testdataFS embed.FS

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

			tmpDir, err := os.MkdirTemp("", "vifstab-test-rcs-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "fstab")
			backupDir := filepath.Join(tmpDir, "backup")

			if len(initialContent) > 0 {
				if err := os.WriteFile(testFile, initialContent, 0644); err != nil {
					t.Fatalf("failed to write initial file: %v", err)
				}
			}

			cfg := config.Config{
				FilePath:  testFile,
				BackupDir: backupDir,
			}

			if err := SaveAndVersion(cfg, newContent); err != nil {
				t.Fatalf("SaveAndVersion failed: %v", err)
			}

			got, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("failed to read updated file: %v", err)
			}
			if string(got) != expectedContent {
				t.Errorf("expected %q, got %q", expectedContent, string(got))
			}

			_, err = os.Stat(backupDir)
			gotRcsExists := !os.IsNotExist(err)
			if gotRcsExists != expectedRcsExists {
				t.Errorf("expected RCS dir exists=%v, got %v", expectedRcsExists, gotRcsExists)
			}
		})
	}
}
