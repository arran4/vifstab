package builtin_test

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/format"
	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/core"
	"github.com/arran4/vifstab/internal/rules/options"
	"golang.org/x/tools/txtar"
)

func TestSystemdSharedRuleRunsOnceAcrossRegistry(t *testing.T) {
	for _, fsName := range []string{"ext4", "vfat", "tmpfs", "nfs", "xfs", "btrfs", "f2fs", "jfs", "nilfs2", "romfs", "cramfs", "bfs", "sshfs", "swap", "none", "futurefs"} {
		t.Run(fsName, func(t *testing.T) {
			cache := rules.NewCache()
			cache.Set(options.SystemdVersionKey, options.TargetSystemdVersion{Major: 262})
			cache.Set("proc_version", "Linux version 6.5.0")
			cache.Set("proc_filesystems", map[string]bool{fsName: true})
			cache.Set("block_devices", rules.BlockDeviceInventory{Available: false})
			api := rules.NewCacheFS(cache, fstest.MapFS{})
			ctx := rules.WithCache(context.Background(), cache)
			raw := "UUID=example /target " + fsName + " x-systemd.mount-timeout=bad,x-systemd.requires=,x-systemd.requires=second.mount,x-systemd.future=value 0 2"
			entries, err := core.ParseFstab(strings.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			original := append([]string(nil), entries[0].Fields...)
			var systemdIssues []rules.ValidationIssue
			for _, rule := range rules.GetAllRules() {
				issues, err := rule.Validate(ctx, api, entries[0])
				if err != nil {
					t.Fatalf("rule %s: %v", rule.Name(), err)
				}
				for _, issue := range issues {
					if strings.HasPrefix(issue.Message, "systemd option") || strings.HasPrefix(issue.Message, "Unverified systemd extension") {
						systemdIssues = append(systemdIssues, issue)
					}
				}
			}
			if len(systemdIssues) != 3 {
				t.Fatalf("missing/duplicate shared checks: %v", systemdIssues)
			}
			if !strings.Contains(systemdIssues[0].Message, "mount-timeout") || !strings.Contains(systemdIssues[1].Message, "requires") || !strings.Contains(systemdIssues[2].Message, "future") {
				t.Errorf("systemd findings reordered: %v", systemdIssues)
			}
			if !reflect.DeepEqual(original, entries[0].Fields) || entries[0].RawLine != raw {
				t.Error("validation changed source options")
			}
		})
	}
}

func TestSystemdTXTAROptionsSurviveFormat(t *testing.T) {
	files, err := embeddedTxtarFS.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "systemd_options_") {
			continue
		}
		count++
		data, err := embeddedTxtarFS.ReadFile("testdata/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, section := range txtar.Parse(data).Files {
			if section.Name != "input" {
				continue
			}
			original, err := core.ParseFstab(bytes.NewReader(section.Data))
			if err != nil {
				t.Fatal(err)
			}
			var formatted bytes.Buffer
			if err := format.Format(bytes.NewReader(section.Data), &formatted); err != nil {
				t.Fatal(err)
			}
			after, err := core.ParseFstab(&formatted)
			if err != nil || len(after) != len(original) {
				t.Fatalf("format parse %s: %v", file.Name(), err)
			}
			for i := range original {
				if original[i].Fields[3] != after[i].Fields[3] {
					t.Errorf("%s line%d option spelling/order changed", file.Name(), i+1)
				}
			}
		}
	}
	if count != 5 {
		t.Fatalf("expected all five systemd TXTAR fixtures, got %d", count)
	}
}
