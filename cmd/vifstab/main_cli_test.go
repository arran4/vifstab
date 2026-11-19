package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/editor"
	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/txtar"
)

func runCLI(t *testing.T, bin, input string, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Stdin = strings.NewReader(input)
	// Offline verification must work without host backup/cache or temporary files.
	cmd.Env = append(os.Environ(), "HOME=/nonexistent-vifstab-home", "XDG_CACHE_HOME=/nonexistent-vifstab-cache", "TMPDIR=/nonexistent-vifstab-tmp", "EDITOR=/nonexistent-editor")
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return code, out.String(), errout.String()
}
func TestTxtarSafety(t *testing.T) {
	bin := buildBinary(t)
	for _, args := range [][]string{
		{"--snapshot", "-", "edit", "--backup-dir", "/tmp"},
		{"verify", "--snapshot", "-", "--file", "/etc/fstab"},
		{"verify", "--snapshot", "-", "--backup-dir", "/tmp"},
		{"verify", "--snapshot", "-", "--systemd-reload", "auto"},
		{"verify", "--snapshot", "-", "--capture", "-"},
		{"verify", "--snapshot", "-", "--static"},
		{"verify", "--snapshot", "-", "--unredacted"},
		{"verify", "--snapshot", ""},
		{"verify", "--capture", ""},
		{"edit", "--snapshot", "-"},
		{"diagram", "--snapshot", "-"},
		{"uefi", "--snapshot", "-"},
		{"verify", "--snapshot", "-", "diagram"},
	} {
		if code, _, _ := runCLI(t, bin, "", args...); code == 0 {
			t.Fatalf("unsafe options accepted: %v", args)
		}
	}
}
func TestReplayCommand(t *testing.T) {
	bin := buildBinary(t)
	for _, tc := range []struct {
		args              []string
		input, diagnostic string
	}{
		{[]string{"verify", "--snapshot", "/nonexistent-vifstab-snapshot"}, "", "failed to open txtar file"},
		{[]string{"verify", "--snapshot", "-"}, "NOT A TXTAR", "missing fstab data"},
		{[]string{"verify", "--snapshot", "-"}, strings.Repeat("a", txtar.MaxArchiveSize+1), "exceeds limit"},
	} {
		code, _, errout := runCLI(t, bin, tc.input, tc.args...)
		if code == 0 || !strings.Contains(errout, tc.diagnostic) {
			t.Fatalf("args %v: code=%d stderr=%q", tc.args, code, errout)
		}
	}
}
func TestReplayProductionFindingsAndExitParity(t *testing.T) {
	bin := buildBinary(t)
	for _, tc := range []struct {
		name, text, finding string
		wantCode            int
	}{
		{"valid", "tmpfs /tmp tmpfs defaults 0 0\n", "Validation successful.", 0},
		{"invalid", "UUID=PRIVATE_UUID /boot/efi ext4 defaults 0 2\n", "GPT partition type is EFI System Partition (ESP) but filesystem is not FAT", 1},
		{"kernel", "/dev/PRIVATE_DEVICE /tmp btrfs compress=zstd 0 2\n", "4.14", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := rules.NewCache()
			c.Set("proc_version", "Linux version 4.13.0")
			c.Set("proc_filesystems", map[string]bool{"ext4": true, "btrfs": true, "tmpfs": true})
			c.Set("block_devices", rules.BlockDeviceInventory{Available: true, Devices: []rules.BlockDevice{{Path: "/dev/PRIVATE_DEVICE", UUID: "PRIVATE_UUID", FSType: "ext4", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b"}}})
			api := rules.NewCacheFS(c, fstest.MapFS{})
			api.OfflineReplay = true
			findings, err := editor.VerifyAndLintReader(strings.NewReader(tc.text), api)
			if err != nil {
				t.Fatal(err)
			}
			blocked := false
			for _, v := range findings {
				blocked = blocked || rules.IsBlockingIssue(v)
			}
			if blocked != (tc.wantCode == 1) {
				t.Fatalf("production findings do not demonstrate expected status: %#v", findings)
			}
			inputPath := filepath.Join(t.TempDir(), "fstab")
			if err := os.WriteFile(inputPath, []byte(tc.text), 0600); err != nil {
				t.Fatal(err)
			}
			for _, raw := range []bool{true, false} {
				data, err := txtar.CaptureSystem(context.Background(), api, inputPath, raw)
				if err != nil {
					t.Fatal(err)
				}
				snapshotPath := filepath.Join(t.TempDir(), "snapshot.txtar")
				if err := os.WriteFile(snapshotPath, data, 0600); err != nil {
					t.Fatal(err)
				}
				var firstOut, firstErr string
				for i, args := range [][]string{{"verify", "--snapshot", snapshotPath}, {"verify", "-snapshot", snapshotPath}, {"verify", "--snapshot", "-"}, {"verify", "-snapshot", "-"}} {
					code, out, errout := runCLI(t, bin, string(data), args...)
					if code != tc.wantCode || !strings.Contains(out, tc.finding) {
						t.Fatalf("%v: exit=%d stdout=%q stderr=%q", args, code, out, errout)
					}
					if raw {
						for _, v := range findings {
							if !strings.Contains(out, fmt.Sprintf("[%s] %s (Line: %s)", v.Level, v.Message, v.Line)) {
								t.Fatalf("production finding missing from CLI: %#v\n%s", v, out)
							}
						}
					}
					if i == 0 {
						firstOut, firstErr = out, errout
					} else if firstOut != out || firstErr != errout {
						t.Fatal("replay entry points disagree")
					}
				}
			}
		})
	}
}
func TestCaptureCommand(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "fstab")
	output := filepath.Join(dir, "snapshot.txtar")
	text := "# PRIVATE_HOST\n\ntmpfs /mnt/PRIVATE_TARGET tmpfs defaults 0 0\n"
	if err := os.WriteFile(input, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI(t, bin, "", "verify", "--static", "--file", input, "--capture", output)
	if code != 0 || !strings.Contains(stdout, "Validation successful.") {
		t.Fatalf("capture: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("permissions=%v", info.Mode().Perm())
	}
	before, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(before), "PRIVATE_") {
		t.Fatal("private fstab values leaked")
	}
	if _, err := txtar.ParseTXTAR(before); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runCLI(t, bin, "", "verify", "--static", "--file", input, "--capture", output); code == 0 {
		t.Fatal("existing output overwritten")
	}
	after, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("existing output changed")
	}
	link := filepath.Join(dir, "symlink.txtar")
	if err := os.Symlink(output, link); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runCLI(t, bin, "", "verify", "--static", "--file", input, "--capture", link); code == 0 {
		t.Fatal("symlink output followed")
	}
	if code, _, _ := runCLI(t, bin, "", "verify", "--static", "--file", input, "--capture", filepath.Join(dir, "missing-dir", "snapshot")); code == 0 {
		t.Fatal("output error ignored")
	}
	code, stdout, stderr = runCLI(t, bin, "", "verify", "--static", "--file", input, "--capture", "-", "--unredacted")
	if code != 0 || !strings.Contains(stdout, "PRIVATE_HOST") || !strings.Contains(stderr, "Warning: unredacted") {
		t.Fatalf("unredacted stdout capture: %d %q %q", code, stdout, stderr)
	}
	if _, err := txtar.ParseTXTAR([]byte(stdout)); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runCLI(t, bin, "", "verify", "--static", "--file", input, "--capture", "-")
	if code != 0 || strings.Contains(stdout, "PRIVATE_") {
		t.Fatalf("redacted stdout capture: %d %q %q", code, stdout, stderr)
	}
	if _, err := txtar.ParseTXTAR([]byte(stdout)); err != nil {
		t.Fatal(err)
	}
	if original, err := os.ReadFile(input); err != nil || string(original) != text {
		t.Fatal("capture changed source fstab")
	}
}

// An operational parser failure must not discard a bounded unredacted support
// case. Both capture verification and subsequent replay return failure.
func TestCaptureRetainsTechnicalValidationFailure(t *testing.T) {
	bin := buildBinary(t)
	output := filepath.Join(t.TempDir(), "case.txtar")
	input := strings.Repeat("a", 70*1024) + "\n"
	code, _, stderr := runCLI(t, bin, input, "verify", "--static", "--file", "-", "--capture", output, "--unredacted")
	if code != 1 || !strings.Contains(stderr, "capture saved; verification failed") {
		t.Fatalf("code=%d, stderr=%q", code, stderr)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := txtar.ParseTXTAR(data)
	if err != nil || string(snapshot.Fstab) != input {
		t.Fatalf("capture lost failed input: %v", err)
	}
	code, _, stderr = runCLI(t, bin, string(data), "verify", "--snapshot", "-")
	if code != 1 || !strings.Contains(stderr, "failed to parse") {
		t.Fatalf("replay code=%d, stderr=%q", code, stderr)
	}
}
