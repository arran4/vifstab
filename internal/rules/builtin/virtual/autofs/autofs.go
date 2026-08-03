package autofs

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func init() {
	rules.Register(&AutofsRule{})
}

type AutofsRule struct{}

func (r *AutofsRule) Name() string {
	return "autofs"
}

func (r *AutofsRule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *AutofsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) < 3 || entry.Fields[2] != "autofs" {
		return nil, nil
	}

	fsOptions := options.MergeOptions(
		options.CommonOptions(),
		map[string]options.OptionValidator{
			"fd":           options.IntOption{},
			"pgrp":         options.IntOption{},
			"minproto":     options.IntOption{},
			"maxproto":     options.IntOption{},
			"indirect":     options.NoValueOption{},
			"direct":       options.NoValueOption{},
			"offset":       options.NoValueOption{},
			"strictexpire": options.NoValueOption{},
			"uid":          options.UIDOption{},
			"gid":          options.GIDOption{},
			"ignore":       options.NoValueOption{},
		},
	)

	return options.ValidateOptionsMap(ctx, api, entry, "autofs", fsOptions), nil
}
