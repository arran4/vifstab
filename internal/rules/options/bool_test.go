package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestBoolOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 12}
	opt := options.BoolOption{}

	validValues := []string{"0", "1", "yes", "no", "true", "false", ""}
	for _, val := range validValues {
		t.Run("valid value "+val, func(t *testing.T) {
			hasVal := val != ""
			issues := opt.ValidateOption(ctx, api, entry, "sysfs", "show", val, hasVal)
			if len(issues) != 0 {
				t.Errorf("expected 0 issues for %q, got %d", val, len(issues))
			}
		})
	}

	t.Run("invalid boolean value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "sysfs", "show", "maybe", true)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Invalid value 'maybe' for sysfs mount option 'show' on line 12, must be a boolean (0, 1, yes, no, true, false)"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
