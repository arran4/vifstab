package hfs

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type HfsRule struct{}

func init() {
	rules.Register(&HfsRule{})
}

func (r *HfsRule) Name() string {
	return "hfs"
}

func (r *HfsRule) Dependencies() []string {
	return []string{}
}

func (r *HfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "hfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "hfs", hfsOptions), nil
}

var hfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
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
					Message: fmt.Sprintf("Invalid value '%s' for hfs mount option '%s' on line %d, must be 4 characters long", val, key, entry.LineNum),
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
					Message: fmt.Sprintf("Invalid value '%s' for hfs mount option '%s' on line %d, must be 4 characters long", val, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"uid":        options.UIDOption{},
	"gid":        options.GIDOption{},
	"dir_umask":  options.OctalOption{},
	"file_umask": options.OctalOption{},
	"umask":      options.OctalOption{},
	"session":    options.IntOption{},
	"part":       options.IntOption{},
	"quiet":      options.NoValueOption{},
	"codepage":   options.CodepageOption{},
	"iocharset":  options.IOCharsetOption{},
})
