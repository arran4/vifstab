package hfsplus

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type HfsPlusRule struct{}

func init() {
	rules.Register(&HfsPlusRule{})
}

func (r *HfsPlusRule) Name() string {
	return "hfsplus"
}

func (r *HfsPlusRule) Dependencies() []string {
	return []string{}
}

func (r *HfsPlusRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "hfsplus" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "hfsplus", hfsPlusOptions), nil
}

var hfsPlusOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"creator": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if len(val) != 4 {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for hfsplus mount option '%s' on line %d, must be 4 characters long", val, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"type": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if len(val) != 4 {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for hfsplus mount option '%s' on line %d, must be 4 characters long", val, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"uid":         options.UIDOption{},
	"gid":         options.GIDOption{},
	"umask":       options.OctalOption{},
	"session":     options.IntOption{},
	"part":        options.IntOption{},
	"decompose":   options.NoValueOption{},
	"nodecompose": options.NoValueOption{},
	"force":       options.NoValueOption{},
	"nls":         options.IOCharsetOption{}, // We can use IOCharsetOption to validate charset
})
