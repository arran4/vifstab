package rules_test

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin" // Register builtins
	"github.com/arran4/vifstab/internal/rules/core"
	"golang.org/x/tools/txtar"
)

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

func WalkFiles(root fs.FS, dir string) ([]string, error) {
	var files []string
	err := fs.WalkDir(root, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, p)
		return nil
	})
	sort.Strings(files)
	return files, err
}

func TestRules(t *testing.T) {
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
			inputFS, expectedFS, optionsRaw := SplitInputExpected(ar)

			// Parse options
			var options struct {
				Rules []string `json:"rules"`
			}
			if optionsRaw != "" {
				if err := json.Unmarshal([]byte(optionsRaw), &options); err != nil {
					t.Fatalf("parse options: %v", err)
				}
			}

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
			procData, err := fs.ReadFile(inputFS, "proc/filesystems")
			if err == nil {
				fsMap := make(map[string]bool)
				lines := strings.Split(string(procData), "\n")
				for _, line := range lines {
					fields := strings.Fields(line)
					if len(fields) > 0 {
						fsMap[fields[len(fields)-1]] = true
					}
				}
				c.Set("proc_filesystems", fsMap)
			}

			ctx := rules.WithCache(context.Background(), c)

			allRules := rules.GetAllRules()
			var rulesToRun []rules.Rule

			if len(options.Rules) > 0 {
				ruleMap := make(map[string]rules.Rule)
				for _, r := range allRules {
					ruleMap[r.Name()] = r
				}
				for _, ruleName := range options.Rules {
					if r, ok := ruleMap[ruleName]; ok {
						rulesToRun = append(rulesToRun, r)
					} else {
						t.Fatalf("unknown rule specified in options: %s", ruleName)
					}
				}
			} else {
				rulesToRun = allRules
			}

			var allSuggestions []rules.Suggestion
			for _, entry := range entries {
				for _, r := range rulesToRun {
					suggestions := r.Validate(ctx, entry)
					allSuggestions = append(allSuggestions, suggestions...)
				}
			}

			// Construct got output
			var gotOutput string
			if len(allSuggestions) > 0 {
				gotOutput += "Validation errors found:\n"
				for _, s := range allSuggestions {
					gotOutput += fmt.Sprintf(" - %s (Line: %s)\n", s.Message, s.Line)
				}
			}

			wantOutputBytes, err := fs.ReadFile(expectedFS, "output.txt")
			if err != nil && len(allSuggestions) > 0 {
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
