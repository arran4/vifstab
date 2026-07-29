package devpts_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/virtual/devpts"

	"embed"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/arran4/vifstab/internal/rules/core"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/*.txtar
var testdataFS embed.FS

func TestDevptsTxtarRules(t *testing.T) {
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
				if file.Name == "input" || file.Name == "input/etc/fstab" {
					inputData = string(file.Data)
				} else if file.Name == "issues" || file.Name == "expected/validationerrors.txt" {
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
					c.Set(rules.CacheKey(k), v)
				}
			}

			ctx := rules.WithCache(context.Background(), c)
			mockFs := fstest.MapFS{}
			for _, file := range ar.Files {
				if strings.HasPrefix(file.Name, "input/") && file.Name != "input/etc/fstab" {
					mockFs[strings.TrimPrefix(file.Name, "input/")] = &fstest.MapFile{
						Data: file.Data,
					}
				}
			}

			rule := &devpts.DevptsRule{}
			var allIssues []rules.ValidationIssue
			var validateErr error

			for _, entry := range entries {
				issues, err := rule.Validate(ctx, rules.NewCacheFS(c, mockFs), entry)
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

			wantOutput := strings.TrimSpace(expectedIssues)
			gotOutput = strings.TrimSpace(gotOutput)
			if gotOutput != wantOutput {
				t.Fatalf("mismatch output\nwant:\n%s\ngot:\n%s", wantOutput, gotOutput)
			}
		})
	}
}
