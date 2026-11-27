package editor

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin"
)

func TestVerifyAndLint_EmptyWorldWritableTarget(t *testing.T) {
	dir := t.TempDir()
	fstab := filepath.Join(dir, "empty_fstab")
	if err := os.WriteFile(fstab, []byte("# empty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(fstab, 0666); err != nil {
		t.Fatal(err)
	}

	issues, err := VerifyAndLint(config.Config{FilePath: fstab}, rules.ActualFS{})
	if err != nil {
		t.Fatalf("VerifyAndLint error: %v", err)
	}
	found := false
	for _, iss := range issues {
		if strings.Contains(iss.Message, "world-writable") {
			found = true
			if iss.Level != rules.IssueLevelWarning {
				t.Fatalf("got level %v, want warning", iss.Level)
			}
		}
	}
	if !found {
		t.Fatalf("expected fstab-permissions world-writable warning on empty fstab, got %+v", issues)
	}
}

func TestEditAndLint_InspectsEffectiveTargetFile(t *testing.T) {
	dir := t.TempDir()
	fstab := filepath.Join(dir, "alt_fstab")
	if err := os.WriteFile(fstab, []byte("# empty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(fstab, 0666); err != nil {
		t.Fatal(err)
	}

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	cfg := config.Config{FilePath: fstab}
	deps := Dependencies{
		Editor: func(path string) error { return nil },
		Input:  strings.NewReader("q\n"),
	}

	editErr := EditAndLint(cfg, rules.ActualFS{}, deps)
	_ = w.Close()
	var outBuf bytes.Buffer
	_, _ = io.Copy(&outBuf, r)
	_ = r.Close()

	if editErr != nil {
		t.Fatalf("EditAndLint error: %v", editErr)
	}
	if !strings.Contains(outBuf.String(), "world-writable") {
		t.Fatalf("expected world-writable warning for %s on stdout, got:\n%s", fstab, outBuf.String())
	}
}
