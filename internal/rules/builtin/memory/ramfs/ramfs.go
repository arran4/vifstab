package ramfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type RamfsRule struct{}

func init() {
	rules.Register(&RamfsRule{})
}

func (r *RamfsRule) Name() string {
	return "ramfs"
}

func (r *RamfsRule) Dependencies() []string {
	return []string{}
}

func (r *RamfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ramfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "ramfs", ramfsOptions), nil
}

var ramfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"mode": options.OctalOption{},
})
