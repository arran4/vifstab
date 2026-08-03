package fuse

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type FuseRule struct{}

func init() {
	rules.Register(&FuseRule{})
}

func (r *FuseRule) Name() string {
	return "fuse"
}

func (r *FuseRule) Dependencies() []string {
	return []string{}
}

func (r *FuseRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "fuse" {
		return nil, nil
	}

	fsOpts := options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
		"default_permissions": options.NoValueOption{},
		"allow_other":         options.NoValueOption{},
		"rootmode":            options.UmaskOption{},
		"blkdev":              options.NoValueOption{},
		"blksize":             options.IntOption{},
		"max_read":            options.SizeOption{},
		"fd":                  options.IntOption{},
		"user_id":             options.UIDOption{},
		"group_id":            options.GIDOption{},
		"fsname":              options.StringOption{},
		"subtype":             options.StringOption{},

		"allow_root":   options.NoValueOption{},
		"auto_unmount": options.NoValueOption{},

		"kernel_cache":     options.NoValueOption{},
		"auto_cache":       options.NoValueOption{},
		"umask":            options.UmaskOption{},
		"fmask":            options.UmaskOption{},
		"dmask":            options.UmaskOption{},
		"uid":              options.UIDOption{},
		"gid":              options.GIDOption{},
		"entry_timeout":    options.StringOption{},
		"negative_timeout": options.StringOption{},
		"attr_timeout":     options.StringOption{},
		"ac_attr_timeout":  options.StringOption{},
		"noforget":         options.NoValueOption{},
		"remember":         options.StringOption{},
		"modules":          options.StringOption{},

		"setuid":          options.StringOption{},
		"drop_privileges": options.NoValueOption{},

		"from_code": options.IOCharsetOption{},
		"to_code":   options.IOCharsetOption{},

		"subdir":     options.StringOption{},
		"rellinks":   options.NoValueOption{},
		"norellinks": options.NoValueOption{},
	})

	return options.ValidateOptionsMap(ctx, api, entry, fsType, fsOpts), nil
}
