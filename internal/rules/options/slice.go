package options

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// OptionsValidator defines the interface for validating an entire slice of parsed mount options.
type OptionsValidator interface {
	ValidateOptions(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, opts []OptionState) []rules.ValidationIssue
}

// ExclusiveGroup ensures that from a given set of keys, at most one occurrence is present in the options.
// This detects both conflicting options (e.g. rw vs ro) and redundant options (e.g. rw,rw).
type ExclusiveGroup struct {
	Keys []string
}

func (e ExclusiveGroup) ValidateOptions(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, opts []OptionState) []rules.ValidationIssue {
	var found []string
	for _, opt := range opts {
		for _, k := range e.Keys {
			if opt.Key == k {
				if opt.HasVal {
					found = append(found, opt.Key+"="+opt.Value)
				} else {
					found = append(found, opt.Key)
				}
				break
			}
		}
	}
	if len(found) > 1 {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Conflicting or redundant options found for %s: %s on line %d", fsName, strings.Join(found, ", "), entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}

// CommonOptionsValidators returns standard slice validators for common options.
func CommonOptionsValidators() []OptionsValidator {
	return []OptionsValidator{
		ExclusiveGroup{Keys: []string{"rw", "ro"}},
		ExclusiveGroup{Keys: []string{"suid", "nosuid"}},
		ExclusiveGroup{Keys: []string{"exec", "noexec"}},
		ExclusiveGroup{Keys: []string{"auto", "noauto"}},
		ExclusiveGroup{Keys: []string{"user", "nouser"}},
		ExclusiveGroup{Keys: []string{"sync", "async"}},
		ExclusiveGroup{Keys: []string{"atime", "noatime"}},
		ExclusiveGroup{Keys: []string{"diratime", "nodiratime"}},
		ExclusiveGroup{Keys: []string{"iversion", "noiversion"}},
		ExclusiveGroup{Keys: []string{"lazytime", "nolazytime"}},
		ExclusiveGroup{Keys: []string{"mand", "nomand"}},
	}
}

// AppendOptionsValidator appends validators to a base slice.
func AppendOptionsValidator(base []OptionsValidator, additional ...OptionsValidator) []OptionsValidator {
	res := make([]OptionsValidator, 0, len(base)+len(additional))
	res = append(res, base...)
	res = append(res, additional...)
	return res
}

// ValidateOptionsSlice validates a slice of OptionsValidator against the entry's options.
func ValidateOptionsSlice(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, validators []OptionsValidator) []rules.ValidationIssue {
	if len(entry.Fields) < 4 {
		return nil
	}

	var issues []rules.ValidationIssue
	set := ParseOptionSet(entry.Fields[3])

	for _, v := range validators {
		if v != nil {
			issues = append(issues, v.ValidateOptions(ctx, api, entry, fsName, set.Ordered())...)
		}
	}
	return issues
}
