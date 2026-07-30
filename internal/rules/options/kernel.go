package options

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

// KernelVersionCase pairs kernel version constraints with an OptionValidator.
// It can be used standalone as a conditional "If" validator, or grouped inside a KernelVersionSwitch.
type KernelVersionCase struct {
	MinVer    kernel.Ver       // Inclusive minimum kernel version required
	MaxVer    kernel.Ver       // Inclusive maximum kernel version allowed
	Validator OptionValidator  // Validator to execute if kernel version falls within bounds
	Level     rules.IssueLevel // Issue level if kernel version requirement fails (default: IssueLevelError)
}

func (c KernelVersionCase) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	hostVer := kernel.GetHostKernelVersion(api)

	if !hostVer.IsZero() {
		matchesMin := c.MinVer.IsZero() || hostVer.AtLeast(c.MinVer.Major, c.MinVer.Minor, c.MinVer.Patch)
		matchesMax := c.MaxVer.IsZero() || hostVer.LessThan(c.MaxVer.Major, c.MaxVer.Minor, c.MaxVer.Patch) || hostVer == c.MaxVer

		level := c.Level
		if level == "" {
			level = rules.IssueLevelError
		}

		if !matchesMin {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("%s mount option '%s' on line %d requires kernel %s+ (host kernel is %s)", fsName, key, entry.LineNum, c.MinVer.String(), hostVer.String()),
					Line:    entry.RawLine,
					Level:   level,
				},
			}
		}

		if !matchesMax {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("%s mount option '%s' on line %d is not supported on kernel > %s (host kernel is %s)", fsName, key, entry.LineNum, c.MaxVer.String(), hostVer.String()),
					Line:    entry.RawLine,
					Level:   level,
				},
			}
		}
	}

	if c.Validator != nil {
		return c.Validator.ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
	}

	return nil
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
