package befs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

var befsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"uid":       options.UIDOption{},
	"gid":       options.GIDOption{},
	"iocharset": options.IOCharsetOption{},
	"debug":     options.NoValueOption{},
})

type BefsRule struct{}

func init() {
	rules.Register(&BefsRule{})
}

func (r *BefsRule) Name() string {
	return "befs"
}

func (r *BefsRule) Dependencies() []string {
	return []string{}
}

func (r *BefsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "befs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, r.Name(), befsOptions), nil
}
