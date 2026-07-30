package hfs

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
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

	return rules.ValidateOptionsMap(ctx, api, entry, "hfs", hfsOptions), nil
}

var hfsOptions = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"creator": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
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
	"type": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
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
	"uid":        rules.UIDOption{},
	"gid":        rules.GIDOption{},
	"dir_umask":  rules.OctalOption{},
	"file_umask": rules.OctalOption{},
	"umask":      rules.OctalOption{},
	"session":    rules.IntOption{},
	"part":       rules.IntOption{},
	"quiet":      rules.NoValueOption{},
})
