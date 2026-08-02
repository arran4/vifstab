package legacy

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

const ProcFilesystemsKey rules.CacheKey = "proc_filesystems"

func CheckFSVersion(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsType string, minVer, maxVer kernel.Ver, neverMerged bool, replacement string, isStillPresent bool) []rules.ValidationIssue {
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
		val, ok := c.Get(ProcFilesystemsKey)
		if !ok {
			// Try to load it if not present
			if f, err := os.Open("/proc/filesystems"); err == nil {
				fsMap := make(map[string]bool)
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					fields := strings.Fields(scanner.Text())
					if len(fields) > 0 {
						fsMap[fields[len(fields)-1]] = true
					}
				}
				_ = f.Close()
				c.Set(ProcFilesystemsKey, fsMap)
				val = fsMap
			}
		}

		if validFS, ok := val.(map[string]bool); ok {
			if validFS[fsType] {
				configured = true
			}
		}
	}

	hostVer := kernel.GetHostKernelVersion(api)

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
			var sb strings.Builder
			sb.WriteString(fsType)
			sb.WriteString(" was removed from upstream Linux in kernel ")
			sb.WriteString(maxVer.String())
			sb.WriteString(" (host is ")
			sb.WriteString(hostVer.String())
			sb.WriteString(") and is not configured")
			if replacement != "" {
				sb.WriteString("; ")
				sb.WriteString(replacement)
			}
			issues = append(issues, rules.ValidationIssue{
				Message: sb.String(),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		} else if minVer.IsZero() && maxVer.IsZero() {
			var sb strings.Builder
			sb.WriteString(fsType)
			sb.WriteString(" is no longer a current upstream option and is not configured")
			if replacement != "" {
				sb.WriteString("; ")
				sb.WriteString(replacement)
			}
			issues = append(issues, rules.ValidationIssue{
				Message: sb.String(),
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
