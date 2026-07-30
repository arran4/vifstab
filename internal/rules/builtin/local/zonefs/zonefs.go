package zonefs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type ZonefsRule struct{}

func init() {
	rules.Register(&ZonefsRule{})
}

func (r *ZonefsRule) Name() string {
	return "zonefs"
}

func (r *ZonefsRule) Dependencies() []string {
	return []string{}
}

func (r *ZonefsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "zonefs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "zonefs", zonefsOptions), nil
}

var zonefsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"errors": options.EnumOption{
		Allowed: map[string]bool{
			"remount-ro":   true,
			"zone-ro":      true,
			"zone-offline": true,
			"repair":       true,
		},
	},
	"explicit-open": options.NoValueOption{},
})
