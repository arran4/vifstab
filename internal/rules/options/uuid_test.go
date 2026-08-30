package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestUUIDOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 10}
	opt := options.UUIDOption{}

	t.Run("valid uuid", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "uuid", "123e4567-e89b-12d3-a456-426614174000", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("invalid uuid format", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "uuid", "not-a-uuid", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Invalid UUID format 'not-a-uuid' for ext4 mount option 'uuid' on line 10"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("missing value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "uuid", "", false, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Missing value for ext4 mount option 'uuid' on line 10"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("empty value string", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "uuid", "", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Missing value for ext4 mount option 'uuid' on line 10"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
