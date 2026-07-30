package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestMergeOptions(t *testing.T) {
	map1 := map[string]options.OptionValidator{
		"ro": options.NoValueOption{},
	}
	map2 := map[string]options.OptionValidator{
		"size": options.StringOption{},
	}

	merged := options.MergeOptions(map1, map2)
	if len(merged) != 2 {
		t.Fatalf("expected merged length 2, got %d", len(merged))
	}
	if _, ok := merged["ro"]; !ok {
		t.Errorf("expected merged to contain 'ro'")
	}
	if _, ok := merged["size"]; !ok {
		t.Errorf("expected merged to contain 'size'")
	}
}

func TestTODOValidator(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 1}
	todoOpt := options.TODOValidator{Comment: "Custom validator coming soon"}

	issues := todoOpt.ValidateOption(ctx, api, entry, "ext4", "custom_opt", "val", true, nil)
	if len(issues) != 0 {
		t.Errorf("expected 0 issues from TODOValidator, got %d", len(issues))
	}
}

func TestValidateOptionsMap(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}

	optsMap := options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
		"size": options.StringOption{},
	})

	t.Run("valid options line", func(t *testing.T) {
		entry := &rules.FstabEntry{
			LineNum: 1,
			RawLine: "tmpfs /tmp tmpfs rw,size=1G 0 0",
			Fields:  []string{"tmpfs", "/tmp", "tmpfs", "rw,size=1G", "0", "0"},
		}
		issues := options.ValidateOptionsMap(ctx, api, entry, "tmpfs", optsMap)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("unknown option error", func(t *testing.T) {
		entry := &rules.FstabEntry{
			LineNum: 2,
			RawLine: "tmpfs /tmp tmpfs rw,fakeopt 0 0",
			Fields:  []string{"tmpfs", "/tmp", "tmpfs", "rw,fakeopt", "0", "0"},
		}
		issues := options.ValidateOptionsMap(ctx, api, entry, "tmpfs", optsMap)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Unknown tmpfs mount option 'fakeopt' on line 2"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("option validator func adapter", func(t *testing.T) {
		customCalled := false
		customOpt := options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
			customCalled = true
			return nil
		})
		customMap := map[string]options.OptionValidator{"custom": customOpt}
		entry := &rules.FstabEntry{
			LineNum: 3,
			RawLine: "tmpfs /tmp tmpfs custom 0 0",
			Fields:  []string{"tmpfs", "/tmp", "tmpfs", "custom", "0", "0"},
		}
		options.ValidateOptionsMap(ctx, api, entry, "tmpfs", customMap)
		if !customCalled {
			t.Errorf("expected custom OptionValidatorFunc to be invoked")
		}
	})
}
