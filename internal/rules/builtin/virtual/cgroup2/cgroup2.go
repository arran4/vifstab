package cgroup2

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type Cgroup2Rule struct{}

func init() {
	rules.Register(&Cgroup2Rule{})
}

func (r *Cgroup2Rule) Name() string {
	return "cgroup2"
}

func (r *Cgroup2Rule) Dependencies() []string {
	return []string{}
}

func (r *Cgroup2Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "cgroup2" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "cgroup2", cgroup2Options), nil
}

var cgroup2Options = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"nsdelegate":                options.NoValueOption{},
	"favordynmods":              options.NoValueOption{},
	"memory_localevents":        options.NoValueOption{},
	"memory_recursiveprot":      options.NoValueOption{},
	"memory_hugetlb_accounting": options.NoValueOption{},
	"pids_localevents":          options.NoValueOption{},
})
