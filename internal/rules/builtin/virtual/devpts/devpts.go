package devpts

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type DevptsRule struct{}

func init() {
	rules.Register(&DevptsRule{})
}

func (r *DevptsRule) Name() string {
	return "devpts"
}

func (r *DevptsRule) Dependencies() []string {
	return []string{}
}

func (r *DevptsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "devpts" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "devpts", devptsOptions), nil
}

var devptsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"uid":         options.UIDOption{},
	"gid":         options.GIDOption{},
	"mode":        options.OctalOption{},
	"ptmxmode":    options.OctalOption{},
	"newinstance": options.NoValueOption{},
})
