package cli

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/cmd"
	"github.com/arran4/vifstab/internal/outcome"
	"github.com/arran4/vifstab/internal/rules"
)

type failingStdin struct{ err error }

func (s failingStdin) Stat() (os.FileInfo, error) { return nil, s.err }

func TestStdinProbe(t *testing.T) {
	for _, mode := range []fs.FileMode{os.ModeCharDevice, os.ModeNamedPipe, 0600} {
		f, err := (fstest.MapFS{"stdin": &fstest.MapFile{Mode: mode}}).Open("stdin")
		if err != nil {
			t.Fatal(err)
		}
		got, err := stdinIsPiped(f)
		_ = f.Close()
		if err != nil || got != (mode&os.ModeCharDevice == 0) {
			t.Fatalf("mode %v: %v, %v", mode, got, err)
		}
	}
	problem := errors.New("metadata unavailable")
	if got, err := stdinIsPiped(failingStdin{problem}); got || !errors.Is(err, problem) || !strings.Contains(err.Error(), "standard input") {
		t.Fatalf("probe failure: %v, %v", got, err)
	}
}

func TestReportSeverity(t *testing.T) {
	for _, level := range []rules.IssueLevel{rules.IssueLevelInfo, rules.IssueLevelWarning, rules.IssueLevelSecurity, rules.IssueLevelError} {
		var out bytes.Buffer
		err := Report(&out, []rules.ValidationIssue{{Level: level, Message: "finding", Line: "example"}})
		blocking := level != rules.IssueLevelInfo
		var exit *cmd.ErrExitCode
		if blocking != errors.As(err, &exit) || (blocking && exit.Code != 1) {
			t.Fatalf("%s: %v", level, err)
		}
		want := "Validation successful.\nIssues:\n"
		if blocking {
			want = "Validation issues found:\n"
		}
		want += " - [" + string(level) + "] finding (Line: example)\n"
		if out.String() != want {
			t.Fatalf("%s: %q != %q", level, out.String(), want)
		}
	}
}

func TestApplicationOutcomeStatus(t *testing.T) {
	for _, code := range []int{0, 1, 2, 3} {
		problem := errors.New("save problem")
		err := applicationError(outcome.New(code, problem))
		var exit *cmd.ErrExitCode
		if !errors.As(err, &exit) || exit.Code != code || !errors.Is(err, problem) {
			t.Fatalf("status %d: %v", code, err)
		}
	}
}

type failedOutput struct {
	after int
	err   error
}

func (w *failedOutput) Write(p []byte) (int, error) {
	if w.after == 0 {
		return 0, w.err
	}
	w.after--
	return len(p), nil
}
func TestReportPropagatesOutputErrors(t *testing.T) {
	problem := errors.New("closed output")
	for _, level := range []rules.IssueLevel{rules.IssueLevelInfo, rules.IssueLevelError} {
		for after := 0; after < 2; after++ {
			err := Report(&failedOutput{after: after, err: problem}, []rules.ValidationIssue{{Level: level, Message: "finding"}})
			if !errors.Is(err, problem) {
				t.Fatalf("level %s, writes %d: %v", level, after, err)
			}
		}
	}
}
