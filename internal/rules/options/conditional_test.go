package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestRequiresOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 7}

	opt := options.RequiresOption{
		Validator:    options.NoValueOption{},
		RequiredKeys: []string{"ro"},
		CustomErrMsg: "XFS mount option 'norecovery' requires 'ro' on line %d",
	}

	t.Run("required option present", func(t *testing.T) {
		allOpts := map[string]options.OptionState{
			"norecovery": {Key: "norecovery"},
			"ro":         {Key: "ro"},
		}
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "norecovery", "", false, allOpts)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("required option missing", func(t *testing.T) {
		allOpts := map[string]options.OptionState{
			"norecovery": {Key: "norecovery"},
		}
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "norecovery", "", false, allOpts)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "XFS mount option 'norecovery' requires 'ro' on line 7"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}

func TestConflictsOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 9}

	opt := options.ConflictsOption{
		Validator:       options.NoValueOption{},
		ConflictingKeys: []string{"rw"},
	}

	t.Run("conflicting option absent", func(t *testing.T) {
		allOpts := map[string]options.OptionState{
			"ro": {Key: "ro"},
		}
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "ro", "", false, allOpts)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("conflicting option present", func(t *testing.T) {
		allOpts := map[string]options.OptionState{
			"ro": {Key: "ro"},
			"rw": {Key: "rw"},
		}
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "ro", "", false, allOpts)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "ext4 mount option 'ro' conflicts with 'rw' on line 9"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}

func TestConditionalOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 15}

	opt := options.ConditionalOption{
		Validator: options.StringOption{},
		Condition: func(allOpts map[string]options.OptionState) bool {
			_, hasRemount := allOpts["remount"]
			roState, hasRo := allOpts["ro"]
			return hasRemount && (!hasRo || roState.Key == "")
		},
		ErrMsg: "jfs mount option 'resize' on line %d requires a read-write remount",
	}

	t.Run("condition satisfied", func(t *testing.T) {
		allOpts := map[string]options.OptionState{
			"remount": {Key: "remount"},
			"rw":      {Key: "rw"},
			"resize":  {Key: "resize", Value: "1000", HasVal: true},
		}
		issues := opt.ValidateOption(ctx, api, entry, "jfs", "resize", "1000", true, allOpts)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("condition failed", func(t *testing.T) {
		allOpts := map[string]options.OptionState{
			"resize": {Key: "resize", Value: "1000", HasVal: true},
		}
		issues := opt.ValidateOption(ctx, api, entry, "jfs", "resize", "1000", true, allOpts)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "jfs mount option 'resize' on line 15 requires a read-write remount"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
