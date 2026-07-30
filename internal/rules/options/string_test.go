package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestStringOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 5, RawLine: "LABEL=root / ext4 rw 0 1"}
	opt := options.StringOption{}

	t.Run("valid string option with value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "label", "root", true)
		if len(issues) != 0 {
			t.Errorf("expected no issues, got %d", len(issues))
		}
	})

	t.Run("missing value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "label", "", false)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Missing value for ext4 mount option 'label' on line 5"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected msg %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}

func TestOptionalStringOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 5}
	opt := options.OptionalStringOption{}

	t.Run("allows empty value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "btrfs", "compress", "", false)
		if len(issues) != 0 {
			t.Errorf("expected no issues, got %d", len(issues))
		}
	})

	t.Run("allows non-empty value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "btrfs", "compress", "zstd", true)
		if len(issues) != 0 {
			t.Errorf("expected no issues, got %d", len(issues))
		}
	})
}
