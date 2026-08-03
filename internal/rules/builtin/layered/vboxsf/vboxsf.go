package vboxsf

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type VboxsfRule struct{}

func init() {
	rules.Register(&VboxsfRule{})
}

var fsOptions = options.MergeOptions(
	options.CommonOptions(),
	map[string]options.OptionValidator{
		// Booleans
		"rw": options.NoValueOption{},
		"ro": options.NoValueOption{},

		// Users/Groups
		"uid": options.UIDOption{},
		"gid": options.GIDOption{},

		// Time to live
		"ttl":       options.IntOption{},
		"dcachettl": options.IntOption{},
		"inodettl":  options.IntOption{},

		// Buffer sizes
		"maxiopages": options.IntOption{},
		"dirbuf":     options.IntOption{},

		// Caching mode
		"cache": options.EnumOption{Allowed: map[string]bool{
			"default":   true,
			"none":      true,
			"strict":    true,
			"read":      true,
			"readwrite": true,
		}},

		// Permissions
		"dmode": options.OctalOption{},
		"fmode": options.OctalOption{},
		"umask": options.UmaskOption{},
		"dmask": options.UmaskOption{},
		"fmask": options.UmaskOption{},

		// Charset
		"iocharset": options.IOCharsetOption{},
		"convertcp": options.IOCharsetOption{},
	},
)

func (r *VboxsfRule) Name() string {
	return "vboxsf"
}

func (r *VboxsfRule) Dependencies() []string {
	return []string{}
}

func (r *VboxsfRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "vboxsf" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "vboxsf", fsOptions), nil
}
