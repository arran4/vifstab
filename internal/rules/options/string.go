package options

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
)

// StringOption validates that an option requires a string value.
type StringOption struct{}

func (StringOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}

// OptionalStringOption allows either no value or any string value.
type OptionalStringOption struct{}

func (OptionalStringOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	return nil
}
