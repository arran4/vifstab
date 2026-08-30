package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestDirOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{
		existingFiles: map[string]bool{
			"/mnt/target":      true,
			"/mnt/escaped dir": true,
		},
	}
	entry := &rules.FstabEntry{LineNum: 15, RawLine: "none /mnt/target tmpfs target=/mnt/target 0 0"}
	opt := options.DirOption{CheckExistence: true}

	t.Run("existing directory", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "target", "/mnt/target", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("existing escaped directory", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "target", `/mnt/escaped\040dir`, true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("non-existing directory", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "target", "/mnt/missing", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Directory '/mnt/missing' specified in tmpfs mount option 'target' does not exist on line 15"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("missing value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "target", "", false, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Missing directory path for tmpfs mount option 'target' on line 15"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
