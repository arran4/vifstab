package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestUIDOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{
		existingUsers: map[int]bool{1000: true},
	}
	entry := &rules.FstabEntry{LineNum: 4}
	opt := options.UIDOption{}

	t.Run("existing UID", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "uid", "1000", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("non-existing UID", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "uid", "9999", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "User with uid 9999 does not exist"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("invalid UID format", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "uid", "notanint", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Invalid value 'notanint' for ext4 mount option 'uid' on line 4, must be an integer"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}

func TestGIDOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{
		existingGroups: map[int]bool{1000: true},
	}
	entry := &rules.FstabEntry{LineNum: 4}
	opt := options.GIDOption{}

	t.Run("existing GID", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "gid", "1000", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("non-existing GID", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "gid", "9999", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Group with gid 9999 does not exist"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}

func TestOptionalUIDOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{existingUsers: map[int]bool{1000: true}}
	entry := &rules.FstabEntry{LineNum: 4}
	opt := options.OptionalUIDOption{}

	t.Run("no value provided", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "uid", "", false, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("valid value provided", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "uid", "1000", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})
}
