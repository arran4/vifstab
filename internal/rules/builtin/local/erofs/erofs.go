package erofs

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type ErofsRule struct{}

func init() {
	rules.Register(&ErofsRule{})
}

func (r *ErofsRule) Name() string {
	return "erofs"
}

func (r *ErofsRule) Dependencies() []string {
	return []string{}
}

func (r *ErofsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "erofs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "erofs", erofsOptions), nil
}

var erofsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"user_xattr":     options.NoValueOption{},
	"nouser_xattr":   options.NoValueOption{},
	"acl":            options.NoValueOption{},
	"noacl":          options.NoValueOption{},
	"cache_strategy": options.EnumOption{Allowed: map[string]bool{"disabled": true, "readahead": true, "readaround": true}},
	"dax": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if hasVal {
			if val != "always" && val != "never" {
				return []rules.ValidationIssue{
					{
						Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					},
				}
			}
		}
		return nil
	}),
	"device":      options.StringOption{},
	"directio":    options.NoValueOption{},
	"domain_id":   options.StringOption{},
	"fsoffset":    options.IntOption{},
	"inode_share": options.NoValueOption{},
})
