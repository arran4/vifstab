package legacy

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

const ProcFilesystemsKey rules.CacheKey = "proc_filesystems"

func CheckFSVersion(ctx context.Context, entry *rules.FstabEntry, fsType string, minVer, maxVer kernel.Ver, neverMerged bool, replacement string, isStillPresent bool) []rules.ValidationIssue {
	if len(entry.Fields) < 3 {
		return nil
	}
	if entry.Fields[2] != fsType {
		return nil
	}

	// Check if configured
	configured := false
	c := rules.CacheFromContext(ctx)
	if c != nil {
		if v, ok := c.Get(ProcFilesystemsKey); ok {
			if validFS, ok := v.(map[string]bool); ok {
				if validFS[fsType] {
					configured = true
				}
			}
		}
	}

	hostVer := kernel.GetHostKernelVersion(c)

	var issues []rules.ValidationIssue

	if !configured {
		if isStillPresent {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("%s is still present in upstream Linux but is not configured in the current kernel", fsType),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
			return issues
		}

		if neverMerged {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("%s was never merged upstream and is not configured in the current kernel", fsType),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
			return issues
		}

		if !minVer.IsZero() && hostVer.LessThan(minVer.Major, minVer.Minor, minVer.Patch) {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("%s requires kernel %s or newer, but host is %s, and it is not configured", fsType, minVer.String(), hostVer.String()),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		} else if !maxVer.IsZero() && hostVer.AtLeast(maxVer.Major, maxVer.Minor, maxVer.Patch) {
			msg := fmt.Sprintf("%s was removed from upstream Linux in kernel %s (host is %s) and is not configured", fsType, maxVer.String(), hostVer.String())
			if replacement != "" {
				msg += fmt.Sprintf("; %s", replacement)
			}
			issues = append(issues, rules.ValidationIssue{
				Message: msg,
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		} else if minVer.IsZero() && maxVer.IsZero() {
			msg := fmt.Sprintf("%s is no longer a current upstream option and is not configured", fsType)
			if replacement != "" {
				msg += fmt.Sprintf("; %s", replacement)
			}
			issues = append(issues, rules.ValidationIssue{
				Message: msg,
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		} else {
			// Default generic fallback if we couldn't match a specific min/max trigger
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("%s is not configured in the current kernel", fsType),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
	} else {
		// Even if configured, maybe it's out of tree or using a patch.
		// If it's officially removed or never merged, we can emit a suggestion/warning
		// that it's relying on a 3rd party module or patch.
		if neverMerged {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("%s was never merged upstream; its presence indicates a 3rd party patch or module is in use", fsType),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelWarning,
			})
		} else if !maxVer.IsZero() && hostVer.AtLeast(maxVer.Major, maxVer.Minor, maxVer.Patch) {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("%s was removed from upstream Linux in kernel %s, but is currently configured (likely via a patch or 3rd party module)", fsType, maxVer.String()),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelWarning,
			})
		}
	}

	return issues
}
