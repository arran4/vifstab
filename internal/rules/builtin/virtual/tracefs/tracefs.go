package tracefs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type TracefsRule struct{}

func init() {
	rules.Register(&TracefsRule{})
}

func (r *TracefsRule) Name() string {
	return "tracefs"
}

func (r *TracefsRule) Dependencies() []string {
	return []string{}
}

var tracefsOptions = options.MergeOptions(
	options.CommonOptions(),
	map[string]options.OptionValidator{
		"uid":  options.UIDOption{},
		"gid":  options.GIDOption{},
		"mode": options.OctalOption{},
	},
)

func (r *TracefsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "tracefs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "tracefs", tracefsOptions), nil
}
