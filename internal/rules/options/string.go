package options

import (
	"context"
	"fmt"
	"regexp"

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

// SELinuxContextOption validates SELinux context strings.
type SELinuxContextOption struct{}

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

	unquotedVal := val
	if len(val) > 0 && (val[0] == '"' || val[len(val)-1] == '"') {
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			unquotedVal = val[1 : len(val)-1]
		} else {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Malformed or unbalanced quotes in %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
	}

	if key == "rootcontext" && unquotedVal == "@target" {
		return nil
	}

	if unquotedVal == "@target" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Special value '@target' is only allowed for 'rootcontext', invalid in %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	parts := regexp.MustCompile(`:`).Split(unquotedVal, 4)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid SELinux context '%s' for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	return nil
}
