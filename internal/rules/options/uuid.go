package options

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/google/uuid"
)

// UUIDOption validates that a mount option is a valid UUID string.
type UUIDOption struct{}

func (UUIDOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	_, err := uuid.Parse(val)
	if err != nil {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid UUID format '%s' for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	return nil
}
