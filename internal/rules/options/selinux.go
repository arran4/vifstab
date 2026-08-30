package options

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// SELinuxContextOption validates SELinux context strings.
type SELinuxContextOption struct{}

// ValidateOption validates that the SELinux context is not empty if a value is provided.
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

	// Basic validation for SELinux context format
	// Typically: user:role:type:range
	parts := strings.Split(val, ":")
	if len(parts) < 3 {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid SELinux context format '%s' for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	for _, part := range parts {
		if part == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid SELinux context format '%s' (empty field) for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
	}

	return nil
}
