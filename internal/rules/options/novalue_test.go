package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestNoValueOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 1, RawLine: "tmpfs /tmp tmpfs ro 0 0"}
	opt := options.NoValueOption{}

	t.Run("valid option with no value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "ro", "", false)
		if len(issues) != 0 {
			t.Errorf("expected no issues, got %d", len(issues))
		}
	})

	t.Run("invalid option with value provided", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "ro", "true", true)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "ext4 mount option 'ro' does not take a value on line 1"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected msg %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("x- prefix options allow values", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "x-custom", "value", true)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues for x- prefix option, got %d", len(issues))
		}
	})
}
