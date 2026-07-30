package options

import (
	"context"
	"fmt"
	"strings"

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
// If validation fails on the target kernel version, it checks alternative cases and appends
// a concise note indicating which kernel versions support the option/value.
type KernelVersionSwitch struct {
	Cases    []KernelVersionCase
	Fallback OptionValidator // Used if no case matches or if host kernel version is unknown (zero)
}

func (k KernelVersionSwitch) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	hostVer := kernel.GetHostKernelVersion(api)

	var activeCase *KernelVersionCase
	var issues []rules.ValidationIssue
	matched := false

	if !hostVer.IsZero() {
		for i := range k.Cases {
			c := &k.Cases[i]
			matchesMin := c.MinVer.IsZero() || hostVer.AtLeast(c.MinVer.Major, c.MinVer.Minor, c.MinVer.Patch)
			matchesMax := c.MaxVer.IsZero() || hostVer.LessThan(c.MaxVer.Major, c.MaxVer.Minor, c.MaxVer.Patch) || hostVer == c.MaxVer
			if matchesMin && matchesMax {
				activeCase = c
				matched = true
				if c.Validator != nil {
					issues = c.Validator.ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
				}
				break
			}
		}
	}

	if !matched && k.Fallback != nil {
		issues = k.Fallback.ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
	}

	// If there are validation issues, find which alternative cases would succeed
	if len(issues) > 0 {
		var workingRanges []string
		for i := range k.Cases {
			c := &k.Cases[i]
			if c == activeCase {
				continue
			}
			if c.Validator != nil {
				altIssues := c.Validator.ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
				if len(altIssues) == 0 {
					workingRanges = append(workingRanges, formatVersionRange(c.MinVer, c.MaxVer))
				}
			}
		}

		if len(workingRanges) > 0 {
			versionNote := fmt.Sprintf(" (supported in %s)", strings.Join(workingRanges, ", "))
			for i := range issues {
				if !strings.Contains(issues[i].Message, "supported in") {
					issues[i].Message += versionNote
				}
			}
		}
	}

	return issues
}

func formatVersionRange(minVer, maxVer kernel.Ver) string {
	hasMin := !minVer.IsZero()
	hasMax := !maxVer.IsZero()

	if hasMin && hasMax {
		return fmt.Sprintf("kernel %s to %s", minVer.String(), maxVer.String())
	}
	if hasMin {
		return fmt.Sprintf("kernel %s+", minVer.String())
	}
	if hasMax {
		return fmt.Sprintf("kernel <= %s", maxVer.String())
	}
	return "other kernel versions"
}
