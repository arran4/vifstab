package options

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// OptionValidator defines the interface for validating mount options.
type OptionValidator interface {
	ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue
}

// OptionValidatorFunc allows plain functions to satisfy OptionValidator.
type OptionValidatorFunc func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue

func (f OptionValidatorFunc) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
	return f(ctx, api, entry, fsName, key, val, hasVal)
}

// MergeOptions merges multiple OptionValidator maps into a single map.
func MergeOptions(maps ...map[string]OptionValidator) map[string]OptionValidator {
	res := make(map[string]OptionValidator)
	for _, m := range maps {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

// ValidateOptionsMap iterates through options in entry.Fields[3] and validates them against optsMap.
func ValidateOptionsMap(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, optsMap map[string]OptionValidator) []rules.ValidationIssue {
	if len(entry.Fields) < 4 {
		return nil
	}

	var issues []rules.ValidationIssue
	parsedOpts := rules.SplitOptions(entry.Fields[3])

	for _, opt := range parsedOpts {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		hasVal := len(parts) > 1
		if hasVal {
			val = parts[1]
		}

		validator, ok := optsMap[key]
		if !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if validator != nil {
			issues = append(issues, validator.ValidateOption(ctx, api, entry, fsName, key, val, hasVal)...)
		}
	}

	return issues
}
