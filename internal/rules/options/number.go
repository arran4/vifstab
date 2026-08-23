package options

import (
	"context"
	"fmt"
	"strconv"

	"github.com/arran4/vifstab/internal/rules"
)

// IntOption validates that an option has an integer value.
type IntOption struct{}

func (i IntOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	if _, err := strconv.ParseInt(val, 10, 64); err != nil {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be an integer", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}

// IntRangeOption validates that an option has an integer value within specified Min/Max bounds.
type IntRangeOption struct {
	Min *int64
	Max *int64
}

// NewIntRange creates an IntRangeOption with both minimum and maximum bounds.
func NewIntRange(min, max int64) IntRangeOption {
	return IntRangeOption{Min: &min, Max: &max}
}

// NewIntMin creates an IntRangeOption with a minimum bound.
func NewIntMin(min int64) IntRangeOption {
	return IntRangeOption{Min: &min}
}

// NewIntMax creates an IntRangeOption with a maximum bound.
func NewIntMax(max int64) IntRangeOption {
	return IntRangeOption{Max: &max}
}

func (i IntRangeOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if issues := (IntOption{}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts); len(issues) > 0 {
		return issues
	}
	num, _ := strconv.ParseInt(val, 10, 64)
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

// OptionalOctalOption validates an optional octal permissions value.
type OptionalOctalOption struct {
	CustomErrFormat string
}

func (o OptionalOctalOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal {
		return nil
	}
	return (OctalOption{CustomErrFormat: o.CustomErrFormat}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
}

// OctalOption validates octal permissions value.
type OctalOption struct {
	CustomErrFormat string
}

func (o OctalOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
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
