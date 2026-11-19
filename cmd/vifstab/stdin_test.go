package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type stdinProbe struct {
	mode os.FileMode
	err  error
}

func (s stdinProbe) Stat() (os.FileInfo, error) {
	if s.err != nil {
		return nil, s.err
	}
	return stdinInfo{s.mode}, nil
}

type stdinInfo struct{ mode os.FileMode }

func (stdinInfo) Name() string        { return "stdin" }
func (stdinInfo) Size() int64         { return 0 }
func (s stdinInfo) Mode() os.FileMode { return s.mode }
func (stdinInfo) ModTime() time.Time  { return time.Time{} }
func (stdinInfo) IsDir() bool         { return false }
func (stdinInfo) Sys() any            { return nil }

func TestStdinIsPiped(t *testing.T) {
	probeErr := errors.New("probe unavailable")
	for _, tc := range []struct {
		name  string
		probe stdinProbe
		piped bool
	}{
		{"terminal", stdinProbe{mode: os.ModeCharDevice}, false},
		{"pipe", stdinProbe{mode: os.ModeNamedPipe}, true},
		{"regular file", stdinProbe{mode: 0600}, true},
		{"stat failure", stdinProbe{err: probeErr}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stdinIsPiped(tc.probe)
			if got != tc.piped || !errors.Is(err, tc.probe.err) {
				t.Fatalf("got (%v, %v), want (%v, %v)", got, err, tc.piped, tc.probe.err)
			}
			if err != nil && !strings.Contains(err.Error(), "failed to inspect standard input") {
				t.Fatalf("missing context: %v", err)
			}
		})
	}
}

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
