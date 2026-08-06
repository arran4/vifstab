package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestExclusiveGroup(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 10, RawLine: "ext4 / ext4 rw,ro 0 0"}

	validator := options.ExclusiveGroup{Keys: []string{"rw", "ro"}}

	t.Run("no conflict", func(t *testing.T) {
		opts := []options.OptionState{
			{Key: "rw"},
			{Key: "relatime"},
		}
		issues := validator.ValidateOptions(ctx, api, entry, "ext4", opts)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("conflict", func(t *testing.T) {
		opts := []options.OptionState{
			{Key: "rw"},
			{Key: "ro"},
		}
		issues := validator.ValidateOptions(ctx, api, entry, "ext4", opts)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Conflicting or redundant options found for ext4: rw, ro on line 10"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("redundant", func(t *testing.T) {
		opts := []options.OptionState{
			{Key: "rw"},
			{Key: "rw"},
		}
		issues := validator.ValidateOptions(ctx, api, entry, "ext4", opts)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Conflicting or redundant options found for ext4: rw, rw on line 10"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("with value", func(t *testing.T) {
		validator := options.ExclusiveGroup{Keys: []string{"quota"}}
		opts := []options.OptionState{
			{Key: "quota"},
			{Key: "quota", Value: "off", HasVal: true},
		}
		issues := validator.ValidateOptions(ctx, api, entry, "gfs2", opts)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Conflicting or redundant options found for gfs2: quota, quota=off on line 10"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
