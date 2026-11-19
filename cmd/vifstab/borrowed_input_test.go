package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Go exec normally supplies /dev/null for absent stdin. Close it inside a child
// to exercise the real generated actions without fabricating a device handle.
func TestBorrowedClosedStdin(t *testing.T) {
	if name := os.Getenv("VIFSTAB_TEST_CLOSED_SOURCE"); name != "" {
		args := map[string][]string{
			"verify":   {"verify", "--file", "-"},
			"snapshot": {"verify", "--snapshot", "-"},
			"capture":  {"verify", "--file", "-", "--capture", os.Getenv("VIFSTAB_TEST_CAPTURE_PATH")},
		}[name]
		if args == nil {
			t.Fatal("unknown child case")
		}
		_ = os.Stdin.Close()
		os.Args = append([]string{"vifstab"}, args...)
		main()
		os.Exit(0)
	}
	for _, name := range []string{"verify", "snapshot", "capture"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			capture := filepath.Join(dir, "support.txtar")
			cmd := exec.Command(os.Args[0], "-test.run=^TestBorrowedClosedStdin$")
			cmd.Env = []string{"VIFSTAB_TEST_CLOSED_SOURCE=" + name, "VIFSTAB_TEST_CAPTURE_PATH=" + capture, "HOME=" + dir, "XDG_CACHE_HOME=" + dir, "PATH=" + dir}
			out, err := cmd.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 || !strings.Contains(string(out), "text input type unknown") || strings.Contains(string(out), "panic:") {
				t.Fatalf("expected operational unknown without panic: %v %s", err, out)
			}
			if _, err := os.Stat(capture); !os.IsNotExist(err) {
				t.Fatalf("unavailable input created capture output: %v", err)
			}
		})
	}
}
