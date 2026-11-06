package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Exercise the real CLI error handler in a child process: Go's exec package
// normally supplies /dev/null for absent stdin, so close it after child startup.
func TestFormatClosedStdin(t *testing.T) {
	if os.Getenv("VIFSTAB_TEST_CLOSED_STDIN") == "1" {
		_ = os.Stdin.Close()
		os.Args = []string{"vifstab", "format"}
		main()
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestFormatClosedStdin$")
	cmd.Env = append(os.Environ(), "VIFSTAB_TEST_CLOSED_STDIN=1")
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Fatalf("expected operational exit 1, got %v: %s", err, out)
	}
	if !strings.Contains(string(out), "failed to inspect standard input") || strings.Contains(string(out), "panic:") {
		t.Fatalf("expected contextual error without panic: %s", out)
	}
}
