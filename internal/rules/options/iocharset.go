package options

import (
	"context"
	"fmt"
	"regexp"

	"github.com/arran4/vifstab/internal/rules"
)

// IOCharsetOption validates iocharset values (NLS translation tables).
type IOCharsetOption struct{}

var iocharsetRegex = regexp.MustCompile(`^[a-zA-Z0-9\-]+$`)

func (IOCharsetOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	if !iocharsetRegex.MatchString(val) {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}
