package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestSourceOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 14, RawLine: "UUID=1234-5678 /mnt ext4 defaults 0 0"}
	opt := options.SourceOption{}

	t.Run("has value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "overlay", "source", "somepath", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("missing value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "overlay", "source", "", false, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Missing source for overlay mount option 'source' on line 14"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("empty value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "overlay", "source", "", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Missing source for overlay mount option 'source' on line 14"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
