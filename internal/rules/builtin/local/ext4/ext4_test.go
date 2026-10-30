package ext4

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules"
)

func TestExt4DataJournalDeprecationDoesNotBlock(t *testing.T) {
	tests := []struct {
		name         string
		kernel       string
		options      string
		wantLevel    rules.IssueLevel
		wantContains string
	}{
		{name: "before deprecation", kernel: "7.2.99", options: "data=journal"},
		{name: "deprecation boundary", kernel: "7.3.0", options: "data=journal", wantLevel: rules.IssueLevelInfo, wantContains: "deprecated"},
		{name: "after deprecation", kernel: "7.4.0", options: "data=journal", wantLevel: rules.IssueLevelInfo, wantContains: "deprecated"},
		{name: "unknown kernel", kernel: "unknown-version", options: "data=journal"},
		{name: "overridden journal", kernel: "7.3.0", options: "data=journal,data=ordered"},
		{name: "ordered", kernel: "7.3.0", options: "data=ordered"},
		{name: "writeback", kernel: "7.3.0", options: "data=writeback"},
		{name: "invalid data mode", kernel: "7.3.0", options: "data=incorrect", wantLevel: rules.IssueLevelError, wantContains: "Invalid value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := rules.NewCache()
			cache.Set("proc_version", "Linux version "+tt.kernel)
			api := rules.NewCacheFS(cache, fstest.MapFS{})
			entry := &rules.FstabEntry{
				RawLine: "/dev/test /mnt/ext4 ext4 " + tt.options + " 0 0",
				LineNum: 1,
				Fields:  []string{"/dev/test", "/mnt/ext4", "ext4", tt.options, "0", "0"},
			}
			issues, err := (&Ext4Rule{}).Validate(context.Background(), api, entry)
			if err != nil {
				t.Fatalf("validation failed: %v", err)
			}
			if tt.wantLevel == "" {
				if len(issues) != 0 {
					t.Fatalf("expected no issues, got: %+v", issues)
				}
				return
			}
			if len(issues) != 1 {
				t.Fatalf("expected one issue, got: %+v", issues)
			}
			if issues[0].Level != tt.wantLevel {
				t.Errorf("issue level = %q; want %q", issues[0].Level, tt.wantLevel)
			}
			if !strings.Contains(issues[0].Message, tt.wantContains) {
				t.Errorf("issue %q does not contain %q", issues[0].Message, tt.wantContains)
			}
			wantBlocking := tt.wantLevel == rules.IssueLevelError
			if got := rules.IsBlockingIssue(issues[0]); got != wantBlocking {
				t.Errorf("IsBlockingIssue(%q) = %t; want %t (verification exit status)", issues[0].Level, got, wantBlocking)
			}
		})
	}
}
