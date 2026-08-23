package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestIntRangeOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 3}
	opt := options.NewIntRange(1, 100)

	t.Run("valid int value within range", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "size", "50", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("invalid non-int string", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "size", "abc", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Invalid value 'abc' for tmpfs mount option 'size' on line 3, must be an integer"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("value below min", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "size", "0", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Value '0' for tmpfs mount option 'size' on line 3 is less than minimum 1"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("value above max", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "size", "101", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Value '101' for tmpfs mount option 'size' on line 3 is greater than maximum 100"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}

func TestOptionalOctalOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 11}
	opt := options.OptionalOctalOption{}

	t.Run("no value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "mkdir", "", false, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("valid octal", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "mkdir", "0755", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("invalid octal", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "mkdir", "888", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Invalid value '888' for ext4 mount option 'mkdir' on line 11, must be an octal integer"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("missing value with equals", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "mkdir", "", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Missing value for ext4 mount option 'mkdir' on line 11"
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
		issues := opt.ValidateOption(ctx, api, entry, "fat", "umask", "077", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("invalid octal string", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "fat", "umask", "089", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Invalid value '089' for fat mount option 'umask' on line 10, must be an octal integer"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
