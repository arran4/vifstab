package options

import (
	"context"
	"fmt"
	"strconv"

	"github.com/arran4/vifstab/internal/rules"
)

// IntOption validates that an option has an integer value.
type IntOption struct {
	Min *int64
	Max *int64
}

func (i IntOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	num, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be an integer", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	if i.Min != nil && num < *i.Min {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Value '%s' for %s mount option '%s' on line %d is less than minimum %d", val, fsName, key, entry.LineNum, *i.Min),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	if i.Max != nil && num > *i.Max {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Value '%s' for %s mount option '%s' on line %d is greater than maximum %d", val, fsName, key, entry.LineNum, *i.Max),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}

// OctalOption validates octal permissions value.
type OctalOption struct {
	CustomErrFormat string
}

func (o OctalOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	if _, err := strconv.ParseUint(val, 8, 32); err != nil {
		msgType := "must be an octal integer"
		if o.CustomErrFormat != "" {
			msgType = o.CustomErrFormat
		}
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, %s", val, fsName, key, entry.LineNum, msgType),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}
