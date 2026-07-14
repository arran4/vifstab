package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallAndRemoveLocalSkill(t *testing.T) {
	// Setup test environment
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Setup local skill source
	sourceDir := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Test Skill"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test Install
	err = InstallSkill(sourceDir, "testskill", "testagent", "user")
	if err != nil {
		t.Fatalf("InstallSkill failed: %v", err)
	}

	// Verify install path
	destDir := filepath.Join(homeDir, ".testagent", "skills", "testskill")
	if _, err := os.Stat(filepath.Join(destDir, "SKILL.md")); os.IsNotExist(err) {
		t.Errorf("Skill file SKILL.md not found in destDir: %s", destDir)
	}

	// Test Remove
	err = RemoveSkill("testskill", "testagent", "user")
	if err != nil {
		t.Fatalf("RemoveSkill failed: %v", err)
	}

	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		t.Errorf("Skill directory still exists after removal: %s", destDir)
	}
}

func TestInstallEmbeddedSkill(t *testing.T) {
	// Setup test environment
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	err := InstallSkill("vifstab", "", "common", "user")
	if err != nil {
		t.Fatalf("InstallSkill embedded failed: %v", err)
	}

	destDir := filepath.Join(homeDir, ".agents", "skills", "vifstab")
	if _, err := os.Stat(filepath.Join(destDir, "SKILL.md")); os.IsNotExist(err) {
		t.Errorf("Skill file SKILL.md not found in destDir: %s", destDir)
	}
}
