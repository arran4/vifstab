package lustre

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/external"
	"github.com/arran4/vifstab/internal/rules/options"
)

type LustreRule struct{}

func init() {
	rules.Register(&LustreRule{})
}

func (r *LustreRule) Name() string {
	return "lustre"
}

func (r *LustreRule) Dependencies() []string {
	return []string{}
}

var lustreOptions = map[string]options.OptionValidator{
	"abort_recov":          options.NoValueOption{},
	"nosvc":                options.NoValueOption{},
	"nomgs":                options.NoValueOption{},
	"exclude":              options.StringOption{},
	"md_stripe_cache_size": options.IntOption{},
	"max_sectors_kb":       options.IntOption{},
	"recovery_time_soft":   options.IntOption{},
	"recovery_time_hard":   options.IntOption{},
	"noscrub":              options.NoValueOption{},
	"flock":                options.NoValueOption{},
	"noflock":              options.NoValueOption{},
	"localflock":           options.NoValueOption{},
	"user_xattr":           options.NoValueOption{},
	"nouser_xattr":         options.NoValueOption{},
	"user_fid2path":        options.NoValueOption{},
	"nouser_fid2path":      options.NoValueOption{},
	"retry":                options.IntOption{},
	"always_ping":          options.NoValueOption{},
	"lazystatfs":           options.NoValueOption{},
	"nolazystatfs":         options.NoValueOption{},
	"verbose":              options.NoValueOption{},
	"noverbose":            options.NoValueOption{},
	"mgssec": options.EnumOption{
		Allowed: map[string]bool{
			"null":    true,
			"plain":   true,
			"gssnull": true,
			"krb5n":   true,
			"krb5a":   true,
			"krb5i":   true,
			"krb5p":   true,
			"skn":     true,
			"ska":     true,
			"ski":     true,
			"skpi":    true,
		},
	},
	"skpath":  options.StringOption{},
	"mgsname": options.StringOption{},
	"mgsnode": options.StringOption{},
}

func (r *LustreRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "lustre" {
		return nil, nil
	}

	var issues []rules.ValidationIssue
	moduleIssues := external.CheckModuleSupport(ctx, fsType, entry, "lustre", "")
	issues = append(issues, moduleIssues...)

	optMap := options.MergeOptions(options.CommonOptions(), lustreOptions)
	optIssues := options.ValidateOptionsMap(ctx, api, entry, "lustre", optMap)
	issues = append(issues, optIssues...)

	return issues, nil
}
