package sysfs

import (
	"context"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"

	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules/core"
	"golang.org/x/tools/txtar"
)

// Ensure the rule is registered and can process entries.
func TestSysfsRuleSimple(t *testing.T) {
	rule := &SysfsRule{}
	entry := &rules.FstabEntry{
		RawLine: "sysfs /sys sysfs rw,nosuid,nodev,noexec,relatime 0 0",
		LineNum: 1,
		Fields:  strings.Fields("sysfs /sys sysfs rw,nosuid,nodev,noexec,relatime 0 0"),
	}

	suggestions, err := rule.Validate(context.Background(), &rules.CacheFS{}, entry)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	for _, s := range suggestions {
		t.Logf("Suggestion: %s", s.Message)
	}
}

//go:embed testdata/*.txtar
var testdataFS embed.FS

func SplitInputExpected(ar *txtar.Archive) (fstest.MapFS, fstest.MapFS, string) {
	input := fstest.MapFS{}
	expected := fstest.MapFS{}
	optionsRaw := ""

	for _, f := range ar.Files {
		switch {
		case strings.HasPrefix(f.Name, "input/"):
			input[strings.TrimPrefix(f.Name, "input/")] = &fstest.MapFile{Data: f.Data}
		case strings.HasPrefix(f.Name, "expected/"):
			expected[strings.TrimPrefix(f.Name, "expected/")] = &fstest.MapFile{Data: f.Data}
		case f.Name == "options.json":
			optionsRaw = string(f.Data)
		}
	}
	return input, expected, optionsRaw
}

func TestSysfsTxtarRules(t *testing.T) {
	var cases []string
	err := fs.WalkDir(testdataFS, "testdata", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".txtar") {
			return nil
		}
		cases = append(cases, p)
		return nil
	})
	if err != nil {
		t.Fatalf("walk cases: %v", err)
	}
	sort.Strings(cases)

	for _, tc := range cases {
		tc := tc
		t.Run(strings.TrimSuffix(filepath.Base(tc), ".txtar"), func(t *testing.T) {
			raw, err := testdataFS.ReadFile(tc)
			if err != nil {
				t.Fatalf("read %s: %v", tc, err)
			}
			ar := txtar.Parse(raw)
			inputFS, expectedFS, _ := SplitInputExpected(ar)

			// Read input fstab
			fstabData, err := fs.ReadFile(inputFS, "etc/fstab")
			if err != nil {
				t.Fatalf("read input/etc/fstab: %v", err)
			}

			entries, err := core.ParseFstab(strings.NewReader(string(fstabData)))
			if err != nil {
				t.Fatalf("parse input/etc/fstab: %v", err)
			}

			// Pre-fill cache (mocking the OS interaction)
			c := rules.NewCache()

			// Check for kernel version in the archive
			kernelVer := "Linux version 6.1.0"
			for _, f := range ar.Files {
				if f.Name == "kernel" {
					kernelVer = strings.TrimSpace(string(f.Data))
				}
			}
			c.Set("proc_version", kernelVer)

			ctx := rules.WithCache(context.Background(), c)

			rule := &SysfsRule{}
			var allIssues []rules.ValidationIssue
			var validateErr error

			for _, entry := range entries {
				issues, err := rule.Validate(ctx, &rules.CacheFS{Cache: c, FS: inputFS}, entry)
				if err != nil {
					validateErr = fmt.Errorf("rule %s failed: %w", rule.Name(), err)
					break
				}
				allIssues = append(allIssues, issues...)
			}

			if validateErr != nil {
				t.Fatalf("unexpected validation err: %v", validateErr)
			}

			var gotOutput string
			if len(allIssues) > 0 {
				for _, s := range allIssues {
					gotOutput += fmt.Sprintf("%s\n", s.Message)
				}
			}

			wantOutputBytes, err := fs.ReadFile(expectedFS, "validationerrors.txt")
			if err != nil && len(allIssues) > 0 {
				t.Fatalf("unexpected validation errors:\n%s", gotOutput)
			} else if err == nil {
				wantOutput := string(wantOutputBytes)
				if gotOutput != wantOutput {
					t.Fatalf("mismatch output\nwant:\n%s\ngot:\n%s", wantOutput, gotOutput)
				}
			}
		})
	}
}
