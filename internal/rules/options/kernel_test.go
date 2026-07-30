package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/kernel"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestKernelVersionSwitch(t *testing.T) {
	ctx := context.Background()
	entry := &rules.FstabEntry{LineNum: 10}

	// Example: hidepid option in procfs
	// Kernel < 5.8: numeric enum (0, 1, 2)
	// Kernel >= 5.8: extended enum (0, 1, 2, invisible, ptraceable)
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

	t.Run("kernel 5.4 rejects 'invisible'", func(t *testing.T) {
		api := &mockAPI{kernelVerStr: "5.4.0"}
		issues := hidepidSwitch.ValidateOption(ctx, api, entry, "proc", "hidepid", "invisible", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue on kernel 5.4 for 'invisible', got %d", len(issues))
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
