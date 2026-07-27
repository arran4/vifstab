package builtin

import (
	"context"
	"embed"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/core"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/*.txtar
var testdataFS embed.FS

func SplitInputExpected(ar *txtar.Archive) (fstest.MapFS, fstest.MapFS) {
	input := fstest.MapFS{}
	expected := fstest.MapFS{}

	for _, f := range ar.Files {
		switch {
		case strings.HasPrefix(f.Name, "input/"):
			input[strings.TrimPrefix(f.Name, "input/")] = &fstest.MapFile{Data: f.Data}
		case strings.HasPrefix(f.Name, "expected/"):
			expected[strings.TrimPrefix(f.Name, "expected/")] = &fstest.MapFile{Data: f.Data}
		}
	}
	return input, expected
}

func TestFSRule(t *testing.T) {
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
			inputFS, expectedFS := SplitInputExpected(ar)

			// Read input fstab
			fstabData, err := fs.ReadFile(inputFS, "etc/fstab")
			if err != nil {
				t.Fatalf("read input/etc/fstab: %v", err)
			}

			entries, err := core.ParseFstab(strings.NewReader(string(fstabData)))
			if err != nil {
				t.Fatalf("parse input/etc/fstab: %v", err)
			}

			// Pre-fill cache
			c := rules.NewCache()
			procData, err := fs.ReadFile(inputFS, "proc/filesystems")
			if err != nil { // Fallback to a sane default if the test doesn't override it
				procData = []byte("nodev\tsysfs\nnodev\ttmpfs\nnodev\tproc\n\text3\n\text4\n\tsquashfs\n\tvfat\n")
			}
			fsMap := make(map[string]bool)
			lines := strings.Split(string(procData), "\n")
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					fsMap[fields[len(fields)-1]] = true
				}
			}
			c.Set(ProcFilesystemsKey, fsMap)
			for _, f := range ar.Files {
				if strings.HasPrefix(f.Name, "input/cache/") {
					key := strings.TrimPrefix(f.Name, "input/cache/")
					val := strings.TrimSpace(string(f.Data))
					if val == "false" {
						c.Set(rules.CacheKey(key), false)
					} else if val == "true" {
						c.Set(rules.CacheKey(key), true)
					} else {
						c.Set(rules.CacheKey(key), f.Data)
					}
				}
			}

			ctx := rules.WithCache(context.Background(), c)
			allRules := rules.GetAllRules()

			var allIssues []rules.ValidationIssue
			for _, entry := range entries {
				for _, r := range allRules {
					suggs, err := r.Validate(ctx, &rules.CacheFS{}, entry)
					if err != nil {
						t.Fatalf("unexpected error during validate for rule %s: %v", r.Name(), err)
					}
					allIssues = append(allIssues, suggs...)
				}
			}

			wantOutputBytes, err := fs.ReadFile(expectedFS, "validationerrors.txt")
			if err != nil { // No expected output provided, meaning it should be valid
				if len(allIssues) > 0 {
					t.Fatalf("expected valid entry to have no issues, got %v for file %s", allIssues, tc)
				}
			} else { // Expected output provided
				expectedStr := strings.TrimSpace(string(wantOutputBytes))
				for _, exp := range strings.Split(expectedStr, "\n") {
					exp = strings.TrimSpace(exp)
					if exp == "" {
						continue
					}
					found := false
					for _, sugg := range allIssues {
						if strings.Contains(sugg.Message, exp) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected to find issue containing '%s' but it was not present. Got: %v", exp, allIssues)
					}
				}
			}
		})
	}
}
