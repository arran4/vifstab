package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arran4/vifstab/internal/config"
)

func TestSaveAndVersion(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "vifstab-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "fstab")
	backupDir := filepath.Join(tmpDir, "backup")

	// Initial content
	initialContent := []byte("initial fstab content")
	if err := os.WriteFile(testFile, initialContent, 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	cfg := config.Config{
		FilePath:  testFile,
		BackupDir: backupDir,
	}

	newContent := []byte("new fstab content")
	if err := SaveAndVersion(cfg, newContent); err != nil {
		t.Fatalf("SaveAndVersion failed: %v", err)
	}

	// Verify the file was updated
	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}
	if string(got) != string(newContent) {
		t.Errorf("expected %q, got %q", string(newContent), string(got))
	}

	// Verify the backup directory exists (RCS was updated)
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Errorf("expected backup directory to exist: %s", backupDir)
	}
}
