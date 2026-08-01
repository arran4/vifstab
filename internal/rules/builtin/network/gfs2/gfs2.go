package gfs2

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type Gfs2Rule struct{}

func init() {
	rules.Register(&Gfs2Rule{})
}

func (r *Gfs2Rule) Name() string {
	return "gfs2"
}

func (r *Gfs2Rule) Dependencies() []string {
	return []string{}
}

func (r *Gfs2Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "gfs2" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "gfs2", gfs2Options), nil
}

// quotaOption handles the quota option, which can be just 'quota' or 'quota=off|account|on'.
type quotaOption struct{}

func (q quotaOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return nil
	}
	allowed := map[string]bool{"off": true, "account": true, "on": true}
	if !allowed[val] {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}

var gfs2Options = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"lockproto":       options.StringOption{},
	"locktable":       options.StringOption{},
	"localcaching":    options.NoValueOption{},
	"localflocks":     options.NoValueOption{},
	"errors":          options.EnumOption{Allowed: map[string]bool{"panic": true, "withdraw": true}},
	"ignore_local_fs": options.NoValueOption{},
	"upgrade":         options.NoValueOption{},
	"acl":             options.NoValueOption{},
	"noacl":           options.NoValueOption{},
	"spectator":       options.NoValueOption{},
	"suiddir":         options.NoValueOption{},
	"nosuiddir":       options.NoValueOption{},
	"quota":           quotaOption{},
	"noquota":         options.NoValueOption{},
	"discard":         options.NoValueOption{},
	"nodiscard":       options.NoValueOption{},
	"barrier":         options.BoolOption{Sets: options.BoolSetNumeric},
	"nobarrier":       options.NoValueOption{},
	"commit":          options.IntOption{},
	"data":            options.EnumOption{Allowed: map[string]bool{"ordered": true, "writeback": true}},
	"meta":            options.NoValueOption{},
	"quota_quantum":   options.IntOption{},
	"statfs_quantum":  options.IntOption{},
	"statfs_percent":  options.IntOption{},
})
