package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/txtar"
)

func TestTextInputExecutableTXTAR(t *testing.T) {
	archive, err := txtar.ParseFile("testdata/textinput.txtar")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name   string
		Args   []string
		Pipe   bool
		Exit   int
		Stdout string
		Stderr string
	}
	if len(archive.Files) != 1 {
		t.Fatal("fixture must have one cases.json section")
	}
	if err := json.Unmarshal(archive.Files[0].Data, &cases); err != nil {
		t.Fatal(err)
	}
	binary := buildBinary(t)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			dir := t.TempDir()
			path, link := filepath.Join(dir, "fstab fixture"), filepath.Join(dir, "source-link")
			const content = "# fixture input\n"
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("fstab fixture", link); err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			replace := strings.NewReplacer("$FILE", path, "$LINK", link, "$DIR", dir)
			args := make([]string, len(tc.Args))
			for i, arg := range tc.Args {
				args[i] = replace.Replace(arg)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, args...)
			cmd.Env = []string{"PATH=" + dir, "HOME=" + dir, "XDG_CACHE_HOME=" + dir, "VISUAL=forbidden-editor", "EDITOR=forbidden-editor"}
			if tc.Pipe {
				cmd.Stdin = strings.NewReader(content)
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err = cmd.Run()
			code := 0
			if err != nil {
				var exit *exec.ExitError
				if !errors.As(err, &exit) {
					t.Fatal(err)
				}
				code = exit.ExitCode()
			}
			if code != tc.Exit || stdout.String() != replace.Replace(tc.Stdout) || stderr.String() != replace.Replace(tc.Stderr) {
				t.Fatalf("exit=%d stdout=%q stderr=%q; expected exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String(), tc.Exit, tc.Stdout, tc.Stderr)
			}
			after, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != content || !os.SameFile(before, after) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
				t.Fatalf("read-only invocation changed fixture: %q %v", data, err)
			}
		})
	}
}
