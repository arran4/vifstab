package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/kernel"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestKernelVersionCaseStandalone(t *testing.T) {
	ctx := context.Background()
	entry := &rules.FstabEntry{LineNum: 7}

	caseOpt := options.KernelVersionCase{
		MinVer:    kernel.Ver{Major: 6, Minor: 1, Patch: 0},
		Validator: options.NoValueOption{},
	}

	t.Run("fails when host kernel below MinVer", func(t *testing.T) {
		api := &mockAPI{kernelVerStr: "5.15.0"}
		issues := caseOpt.ValidateOption(ctx, api, entry, "ext4", "fast_commit", "", false, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "ext4 mount option 'fast_commit' on line 7 requires kernel 6.1.0+ (host kernel is 5.15.0)"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("passes when host kernel meets MinVer", func(t *testing.T) {
		api := &mockAPI{kernelVerStr: "6.5.0"}
		issues := caseOpt.ValidateOption(ctx, api, entry, "ext4", "fast_commit", "", false, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})
}

func TestKernelVersionSwitch(t *testing.T) {
	ctx := context.Background()
	entry := &rules.FstabEntry{LineNum: 10}

	hidepidSwitch := options.KernelVersionSwitch{
		Cases: []options.KernelVersionCase{
			{
				MaxVer: kernel.Ver{Major: 5, Minor: 7, Patch: 999},
				Validator: options.EnumOption{
					Allowed: map[string]bool{"0": true, "1": true, "2": true},
				},
			},
			{
				MinVer: kernel.Ver{Major: 5, Minor: 8, Patch: 0},
				Validator: options.EnumOption{
					Allowed: map[string]bool{"0": true, "1": true, "2": true, "invisible": true, "ptraceable": true},
				},
			},
		},
	}

	t.Run("kernel 5.4 rejects 'invisible' and reports supported version range", func(t *testing.T) {
		api := &mockAPI{kernelVerStr: "5.4.0"}
		issues := hidepidSwitch.ValidateOption(ctx, api, entry, "proc", "hidepid", "invisible", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue on kernel 5.4 for 'invisible', got %d", len(issues))
		}
		expectedMsg := "Invalid value 'invisible' for proc mount option 'hidepid' on line 10 (supported in kernel 5.8.0+)"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("kernel 5.15 accepts 'invisible'", func(t *testing.T) {
		api := &mockAPI{kernelVerStr: "5.15.0"}
		issues := hidepidSwitch.ValidateOption(ctx, api, entry, "proc", "hidepid", "invisible", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues on kernel 5.15 for 'invisible', got %d", len(issues))
		}
	})

	t.Run("fallback used when version unknown", func(t *testing.T) {
		fallbackSwitch := options.KernelVersionSwitch{
			Cases: []options.KernelVersionCase{
				{
					MinVer:    kernel.Ver{Major: 6, Minor: 0, Patch: 0},
					Validator: options.NoValueOption{},
				},
			},
			Fallback: options.StringOption{},
		}
		api := &mockAPI{kernelVerStr: ""}
		issues := fallbackSwitch.ValidateOption(ctx, api, entry, "ext4", "opt", "val", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected fallback StringOption to allow value, got %d issues", len(issues))
		}
	})
}
