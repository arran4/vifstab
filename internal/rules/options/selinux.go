package options

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// SELinuxContextOption validates SELinux context strings.
// A SELinux context generally has the format: user:role:type[:range]
type SELinuxContextOption struct{}

func (s SELinuxContextOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	parts := strings.Split(val, ":")
	if len(parts) < 3 || len(parts) > 4 {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid SELinux context format for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	for _, part := range parts {
		if part == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid SELinux context format (empty field) for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
	}

	return nil
}
