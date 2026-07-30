package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestUmaskOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 5}
	opt := options.UmaskOption{}

	t.Run("valid umask", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "vfat", "umask", "0022", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("invalid umask", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "vfat", "umask", "999", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
	})
}

func TestModeOptions(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 6}

	modeOpt := options.ModeOption{}
	fmodeOpt := options.FileModeOption{}
	dmodeOpt := options.DirModeOption{}

	t.Run("valid modes", func(t *testing.T) {
		if issues := modeOpt.ValidateOption(ctx, api, entry, "fat", "mode", "0755", true, nil); len(issues) != 0 {
			t.Errorf("modeOpt failed: %v", issues)
		}
		if issues := fmodeOpt.ValidateOption(ctx, api, entry, "fat", "fmode", "0644", true, nil); len(issues) != 0 {
			t.Errorf("fmodeOpt failed: %v", issues)
		}
		if issues := dmodeOpt.ValidateOption(ctx, api, entry, "fat", "dmode", "0777", true, nil); len(issues) != 0 {
			t.Errorf("dmodeOpt failed: %v", issues)
		}
	})
}
