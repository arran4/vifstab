package options

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// SELinuxContextOption validates SELinux context strings.
// A valid SELinux context typically follows the format: user:role:type:level
type SELinuxContextOption struct{}

var selinuxContextRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+:[a-zA-Z0-9_]+:[a-zA-Z0-9_]+(:[a-zA-Z0-9_.,:-]+)?$`)

func (SELinuxContextOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	// Remove surrounding quotes if they exist, common in mount options with contexts
	cleanVal := val
	if len(cleanVal) >= 2 && strings.HasPrefix(cleanVal, `"`) && strings.HasSuffix(cleanVal, `"`) {
		cleanVal = cleanVal[1 : len(cleanVal)-1]
	}

	if !selinuxContextRegex.MatchString(cleanVal) {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid SELinux context format for %s mount option '%s' on line %d: %s", fsName, key, entry.LineNum, val),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	return nil
}
