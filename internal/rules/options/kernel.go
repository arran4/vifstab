package options

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

// KernelVersionCase pairs kernel version constraints with an OptionValidator.
type KernelVersionCase struct {
	MinVer    kernel.Ver      // Inclusive minimum kernel version
	MaxVer    kernel.Ver      // Inclusive maximum kernel version
	Validator OptionValidator // Validator to use if kernel version falls within bounds
}

// KernelVersionSwitch selects and executes an OptionValidator based on host kernel version.
type KernelVersionSwitch struct {
	Cases    []KernelVersionCase
	Fallback OptionValidator // Used if no case matches or if host kernel version is unknown (zero)
}

func (k KernelVersionSwitch) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	hostVer := kernel.GetHostKernelVersion(api)

	if !hostVer.IsZero() {
		for _, c := range k.Cases {
			matchesMin := c.MinVer.IsZero() || hostVer.AtLeast(c.MinVer.Major, c.MinVer.Minor, c.MinVer.Patch)
			matchesMax := c.MaxVer.IsZero() || hostVer.LessThan(c.MaxVer.Major, c.MaxVer.Minor, c.MaxVer.Patch) || hostVer == c.MaxVer
			if matchesMin && matchesMax {
				if c.Validator != nil {
					return c.Validator.ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
				}
				return nil
			}
		}
	}

	if k.Fallback != nil {
		return k.Fallback.ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
	}

	return nil
}
