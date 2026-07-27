package ext4_test

import (
	"context"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/local/ext4"

	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/arran4/vifstab/internal/rules/core"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/*.txtar
var testdataFS embed.FS

func TestExt4TxtarRules(t *testing.T) {
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

			inputData := ""
			expectedIssues := ""
			kernelVer := "Linux version 6.5.0"
			cacheVals := map[string]string{}

			for _, file := range ar.Files {
				if file.Name == "input" {
					inputData = string(file.Data)
				} else if file.Name == "issues" {
					expectedIssues = string(file.Data)
				} else if file.Name == "kernel" {
					kernelVer = "Linux version " + strings.TrimSpace(string(file.Data))
				} else if file.Name == "cache" {
					for _, line := range strings.Split(string(file.Data), "\n") {
						line = strings.TrimSpace(line)
						if line == "" {
							continue
						}
						parts := strings.SplitN(line, "=", 2)
						if len(parts) == 2 {
							cacheVals[parts[0]] = parts[1]
						}
					}
				}
			}

			entries, err := core.ParseFstab(strings.NewReader(inputData))
			if err != nil {
				t.Fatalf("parse input: %v", err)
			}

			c := rules.NewCache()
			c.Set("proc_version", kernelVer)
			for k, v := range cacheVals {
				if v == "true" {
					c.Set(rules.CacheKey(k), true)
				} else if v == "false" {
					c.Set(rules.CacheKey(k), false)
				} else {
					if k == "proc_filesystems" {
						// Format it as a map like CheckFSVersion expects
						fsMap := make(map[string]bool)
						for _, fs := range strings.Split(v, ",") {
							fsMap[fs] = true
						}
						c.Set(rules.CacheKey(k), fsMap)
					} else {
						c.Set(rules.CacheKey(k), v)
					}
				}
			}

			ctx := rules.WithCache(context.Background(), c)

			rule := &ext4.Ext4Rule{}
			var allIssues []rules.ValidationIssue
			var validateErr error

			for _, entry := range entries {
				issues, err := rule.Validate(ctx, &rules.CacheFS{Cache: c}, entry)
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
					level := "Suggestion"
					if s.Level == rules.IssueLevelWarning {
						level = "Warning"
					} else if s.Level == rules.IssueLevelError {
						level = "Error"
					}
					gotOutput += fmt.Sprintf("%s: %s\n", level, s.Message)
				}
			}

			wantOutput := expectedIssues
			if gotOutput != wantOutput {
				t.Fatalf("mismatch output\nwant:\n%s\ngot:\n%s", wantOutput, gotOutput)
			}
		})
	}
}
