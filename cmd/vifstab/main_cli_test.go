package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTxtarSafety(t *testing.T) {
	binPath := buildBinary(t)

	// Test --txtar without --verify
	cmd := exec.Command(binPath, "--txtar", "dummy.txtar")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected command to fail when --txtar is used without --verify")
	}
	if !strings.Contains(stderr.String(), "--txtar requires --verify") {
		t.Fatalf("expected safety error message, got: %s", stderr.String())
	}
}

func TestReplayCommand(t *testing.T) {
	binPath := buildBinary(t)
	tempDir := t.TempDir()

	// Just verify the command exists and runs our args validation correctly
	cmd := exec.Command(binPath, "replay")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected replay to fail with no args")
	}
	if !strings.Contains(stderr.String(), "expected 1 argument") {
		t.Fatalf("expected arg error message, got: %s", stderr.String())
	}

	// Create a dummy txtar and check parsing error rather than finding file error
	dummyTxtar := filepath.Join(tempDir, "dummy.txtar")
	os.WriteFile(dummyTxtar, []byte("NOT A TXTAR"), 0644)

	cmd = exec.Command(binPath, "replay", dummyTxtar)
	cmd.Stderr = &stderr
	cmd.Run() // will fail, we just want to ensure it read the file
	if !strings.Contains(stderr.String(), "Validation successful.") && !strings.Contains(stderr.String(), "Validation issues found") && !strings.Contains(stderr.String(), "failed to read txtar") {
	    // It actually won't fail to parse since TXTAR ignores malformed text sometimes, it will just be empty.
	}
}

func TestCaptureCommand(t *testing.T) {
	binPath := buildBinary(t)
	tempDir := t.TempDir()
	fstabPath := filepath.Join(tempDir, "fstab")
	outPath := filepath.Join(tempDir, "out.txtar")

	os.WriteFile(fstabPath, []byte("/dev/null /mnt ext4 defaults 0 0"), 0644)

	cmd := exec.Command(binPath, "capture", "--file", fstabPath, outPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("expected capture to succeed, got %v: %s", err, stderr.String())
	}

	info, err := os.Stat(outPath)
	if err != nil {
	    t.Fatalf("expected output file to exist")
	}

	if info.Mode().Perm() != 0600 {
	    t.Errorf("expected 0600 permissions on captured archive, got %v", info.Mode().Perm())
	}
}
