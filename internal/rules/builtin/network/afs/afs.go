package afs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type AfsRule struct{}

func init() {
	rules.Register(&AfsRule{})
}

func (r *AfsRule) Name() string {
	return "afs"
}

func (r *AfsRule) Dependencies() []string {
	return []string{}
}

func (r *AfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "afs" {
		return nil, nil
	}

	fsOptions := options.MergeOptions(
		options.CommonOptions(),
		map[string]options.OptionValidator{
			"autocell": options.NoValueOption{},
			"dyn":      options.NoValueOption{},
			"flock": options.EnumOption{
				Allowed: map[string]bool{
					"local":   true,
					"openafs": true,
					"strict":  true,
					"write":   true,
				},
			},
		},
	)

	return options.ValidateOptionsMap(ctx, api, entry, "afs", fsOptions), nil
}
