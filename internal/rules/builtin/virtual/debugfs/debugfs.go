package debugfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type DebugfsRule struct{}

func init() {
	rules.Register(&DebugfsRule{})
}

func (r *DebugfsRule) Name() string {
	return "debugfs"
}

func (r *DebugfsRule) Dependencies() []string {
	return []string{}
}

func (r *DebugfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "debugfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "debugfs", debugfsOptions), nil
}

var debugfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"uid":  options.UIDOption{},
	"gid":  options.GIDOption{},
	"mode": options.OctalOption{},
})
