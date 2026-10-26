package builtin_test

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin"
	"github.com/arran4/vifstab/internal/rules/core"
	snapshotar "github.com/arran4/vifstab/internal/txtar"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/*.txtar */*/testdata/*.txtar
var embeddedTxtarFS embed.FS

type txtarTestOptions struct {
	RuleName  string
	KernelVer string
	Format    string
}

func parseOptionsTxt(data string) txtarTestOptions {
	opts := txtarTestOptions{
		Format: "auto",
	}

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.ToLower(strings.TrimSpace(parts[0]))
			v := strings.TrimSpace(parts[1])
			switch k {
			case "rule", "rulename":
				opts.RuleName = v
			case "kernel", "kernelver", "kernel_version":
				if !strings.HasPrefix(v, "Linux version") {
					v = "Linux version " + v
				}
				opts.KernelVer = v
			case "format", "mode":
				opts.Format = strings.ToLower(v)
			}
		}
	}

	return opts
}

func normalizeRuleName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

func TestAllTxtarRules(t *testing.T) {
	var testCases []string
	err := fs.WalkDir(embeddedTxtarFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".txtar") {
			return nil
		}
		testCases = append(testCases, p)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk txtar files: %v", err)
	}
	sort.Strings(testCases)

	if len(testCases) == 0 {
		t.Fatalf("no txtar test cases found")
	}

	defaultFSList := []string{
		"adfs", "affs", "btrfs", "cramfs", "ext2", "ext3", "ext4", "f2fs", "fat",
		"hfs", "hpfs", "iso9660", "jfs", "msdos", "nilfs2", "ntfs", "ntfs3", "ntfs-3g",
		"overlay", "ubifs", "udf", "ufs", "umsdos", "vfat", "xfs",
		"sysfs", "tmpfs", "proc", "devpts", "usbfs", "debugfs", "squashfs",
	}

	for _, tcPath := range testCases {
		tcPath := tcPath
		t.Run(tcPath, func(t *testing.T) {
			raw, err := embeddedTxtarFS.ReadFile(tcPath)
			if err != nil {
				t.Fatalf("failed to read txtar %s: %v", tcPath, err)
			}
			ar := txtar.Parse(raw)

			snapshot, err := snapshotar.ParseTXTAR(raw)
			if err != nil {
				t.Fatalf("parse snapshot: %v", err)
			}
			inputData := string(snapshot.Fstab)
			kernelVer := snapshot.KernelVer
			cacheVals := snapshot.Cache
			mockFs := snapshot.MockFS
			var expectedIssues, optionsData string
			for _, file := range ar.Files {
				switch file.Name {
				case "issues", "expected/validationerrors.txt":
					expectedIssues = string(file.Data)
				case "options.txt", "options":
					optionsData = string(file.Data)
				}
			}

			parsedOpts := parseOptionsTxt(optionsData)
			if parsedOpts.KernelVer != "" {
				kernelVer = parsedOpts.KernelVer
			}
			if kernelVer == "" && snapshot.Manifest == nil {
				if strings.Contains(tcPath, "/legacy/ext3/") {
					kernelVer = "Linux version 3.0.0"
				} else {
					kernelVer = "Linux version 6.5.0"
				}
			}

			entries, err := core.ParseFstab(strings.NewReader(inputData))
			if err != nil {
				t.Fatalf("parse input: %v", err)
			}

			snapshot.KernelVer = kernelVer
			// These are explicit historical test defaults, never production replay
			// defaults. Versioned support captures keep all missing data unknown.
			if snapshot.Manifest == nil {
				_, hasProcFSFile := mockFs["proc/filesystems"]
				_, hasProcFSCache := cacheVals["proc_filesystems"]
				if !hasProcFSFile && !hasProcFSCache {
					snapshot.Cache["proc_filesystems"] = strings.Join(defaultFSList, ",")
				}
			}
			provider, err := snapshotar.ReplaySnapshot(snapshot)
			if err != nil {
				t.Fatalf("snapshot provider: %v", err)
			}
			api := provider.(*rules.CacheFS)
			if snapshot.Manifest == nil {
				api.OfflineReplay = false
			}
			c := api.Cache

			ctx := rules.WithCache(context.Background(), c)

			// Determine which rules to run
			targetRuleName := parsedOpts.RuleName
			if targetRuleName == "" {
				// Infer target rule from path
				dirParts := strings.Split(filepath.ToSlash(tcPath), "/")
				for i, p := range dirParts {
					if p == "testdata" && i > 0 {
						targetRuleName = dirParts[i-1]
						break
					}
				}
			}

			var rulesToRun []rules.Rule
			if targetRuleName == "" || targetRuleName == "testdata" || targetRuleName == "all" || targetRuleName == "fs" || targetRuleName == "builtin" {
				rulesToRun = rules.GetAllRules()
			} else {
				targetNorm := normalizeRuleName(targetRuleName)
				all := rules.GetAllRules()
				for _, r := range all {
					if normalizeRuleName(r.Name()) == targetNorm {
						rulesToRun = append(rulesToRun, r)
					}
				}
				if len(rulesToRun) == 0 {
					rulesToRun = all
				}
			}

			var allIssues []rules.ValidationIssue
			for _, entry := range entries {
				for _, r := range rulesToRun {
					issues, err := r.Validate(ctx, api, entry)
					if err != nil {
						t.Fatalf("rule %s failed on %s: %v", r.Name(), tcPath, err)
					}
					allIssues = append(allIssues, issues...)
				}
			}

			expectedStr := strings.TrimSpace(expectedIssues)

			// Determine format mode
			fmtMode := parsedOpts.Format
			if fmtMode == "auto" {
				if strings.HasPrefix(filepath.ToSlash(tcPath), "testdata/") {
					fmtMode = "contains"
				} else if strings.HasPrefix(expectedStr, "Error:") || strings.HasPrefix(expectedStr, "Warning:") || strings.HasPrefix(expectedStr, "Suggestion:") {
					fmtMode = "level_prefix"
				} else {
					fmtMode = "plain"
				}
			}

			// Format issues output
			var gotOutput string
			if len(allIssues) > 0 {
				for _, s := range allIssues {
					if fmtMode == "level_prefix" {
						level := "Suggestion"
						switch s.Level {
						case rules.IssueLevelWarning:
							level = "Warning"
						case rules.IssueLevelSecurity:
							level = "Security"
						case rules.IssueLevelError:
							level = "Error"
						case rules.IssueLevelInfo:
							level = "Info"
						}
						gotOutput += fmt.Sprintf("%s: %s\n", level, s.Message)
					} else {
						gotOutput += fmt.Sprintf("%s\n", s.Message)
					}
				}
			}

			gotStr := strings.TrimSpace(gotOutput)

			if expectedStr == "" {
				if len(allIssues) > 0 {
					t.Fatalf("expected valid entry (0 issues), got:\n%s", gotStr)
				}
			} else if fmtMode == "contains" {
				for _, expLine := range strings.Split(expectedStr, "\n") {
					expLine = strings.TrimSpace(expLine)
					if expLine == "" {
						continue
					}
					found := false
					for _, sugg := range allIssues {
						if strings.Contains(sugg.Message, expLine) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected to find issue containing '%s' but it was not present. Got:\n%s", expLine, gotStr)
					}
				}
			} else {
				if gotStr != expectedStr {
					t.Fatalf("mismatch output for %s\nwant:\n%s\ngot:\n%s", tcPath, expectedStr, gotStr)
				}
			}
		})
	}
}
