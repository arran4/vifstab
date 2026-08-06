package rootfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type RootfsRule struct{}

func init() {
	rules.Register(&RootfsRule{})
}

func (r *RootfsRule) Name() string {
	return "rootfs"
}

func (r *RootfsRule) Dependencies() []string {
	return []string{}
}

func (r *RootfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "rootfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "rootfs", rootfsOptions), nil
}

var rootfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"mode": options.OctalOption{},
})
