package options

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
)

// BoolOption accepts optional boolean values (0, 1, yes, no, true, false).
type BoolOption struct{}

func (BoolOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
	if hasVal && val != "" {
		if val != "0" && val != "1" && val != "yes" && val != "no" && val != "true" && val != "false" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be a boolean (0, 1, yes, no, true, false)", val, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
	}
	return nil
}
