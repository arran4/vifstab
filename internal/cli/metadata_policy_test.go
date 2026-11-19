package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditMetadataPolicyModes(t *testing.T) {
	for _, policy := range []string{"strict", "supported", "best-effort"} {
		t.Run(policy, func(t *testing.T) {
			dir := t.TempDir()
			fstab := filepath.Join(dir, "fstab")
			backup := filepath.Join(dir, "backup")
			if err := os.WriteFile(fstab, []byte("# initial\n"), 0640); err != nil {
				t.Fatal(err)
			}
			editorScript := filepath.Join(dir, "ed.sh")
			if err := os.WriteFile(editorScript, []byte("#!/bin/sh\nprintf '# edited\\n' >> \"$2\"\n"), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("VISUAL", editorScript)
			t.Setenv("EDITOR", "")

			seen := map[string]bool{"file": true, "backup-dir": true, "metadata-policy": true}
			if err := Edit(fstab, backup, "never", policy, seen); err != nil {
				t.Fatalf("Edit(%s) failed: %v", policy, err)
			}
			got, err := os.ReadFile(fstab)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(got), "# edited") {
				t.Fatalf("expected edited content in %s, got %q", fstab, got)
			}
			info, err := os.Stat(fstab)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0640 {
				t.Fatalf("expected mode 0640 preserved under %s, got %04o", policy, info.Mode().Perm())
			}
		})
	}
}

func TestFormatInPlaceMetadataPolicyModes(t *testing.T) {
	for _, policy := range []string{"strict", "supported", "best-effort"} {
		t.Run(policy, func(t *testing.T) {
			dir := t.TempDir()
			fstab := filepath.Join(dir, "fstab")
			backup := filepath.Join(dir, "backup")
			if err := os.WriteFile(fstab, []byte("UUID=11111111-1111-1111-1111-111111111111   /   ext4   defaults   0   1\n"), 0640); err != nil {
				t.Fatal(err)
			}

			seen := map[string]bool{"file": true, "in-place": true, "backup-dir": true, "metadata-policy": true}
			if err := Format(fstab, true, backup, policy, seen); err != nil {
				t.Fatalf("Format(--in-place, %s) failed: %v", policy, err)
			}
			info, err := os.Stat(fstab)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0640 {
				t.Fatalf("expected mode 0640 preserved under %s, got %04o", policy, info.Mode().Perm())
			}
		})
	}
	if err := Format("fstab", false, "", "best-effort", map[string]bool{"metadata-policy": true}); err == nil {
		t.Fatalf("expected error when --metadata-policy is passed without --in-place")
	}
}
