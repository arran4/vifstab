package coda

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type CodaRule struct{}

func init() {
	rules.Register(&CodaRule{})
}

func (r *CodaRule) Name() string {
	return "coda"
}

func (r *CodaRule) Dependencies() []string {
	return []string{}
}

func (r *CodaRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "coda" {
		return nil, nil
	}

	fsOptions := options.MergeOptions(
		options.CommonOptions(),
		map[string]options.OptionValidator{
			"fd": options.IntOption{},
		},
	)

	return options.ValidateOptionsMap(ctx, api, entry, "coda", fsOptions), nil
}
