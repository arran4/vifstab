package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestEnumOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 8}
	opt := options.EnumOption{
		Allowed: map[string]bool{"continue": true, "remount-ro": true, "panic": true},
	}

	t.Run("valid enum value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "errors", "remount-ro", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("invalid enum value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "errors", "ignore", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Invalid value 'ignore' for ext4 mount option 'errors' on line 8"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("missing value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "errors", "", false, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Missing value for ext4 mount option 'errors' on line 8"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
