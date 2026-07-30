package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestIntOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 3}
	minVal := int64(1)
	maxVal := int64(100)
	opt := options.IntOption{Min: &minVal, Max: &maxVal}

	t.Run("valid int value within range", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "size", "50", true)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("invalid non-int string", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "size", "abc", true)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Invalid value 'abc' for tmpfs mount option 'size' on line 3, must be an integer"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("value below min", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "size", "0", true)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Value '0' for tmpfs mount option 'size' on line 3 is less than minimum 1"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("value above max", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "size", "101", true)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Value '101' for tmpfs mount option 'size' on line 3 is greater than maximum 100"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}

func TestOctalOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 10}
	opt := options.OctalOption{}

	t.Run("valid octal permissions", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "fat", "umask", "077", true)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("invalid octal string", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "fat", "umask", "089", true)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Invalid value '089' for fat mount option 'umask' on line 10, must be an octal integer"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
