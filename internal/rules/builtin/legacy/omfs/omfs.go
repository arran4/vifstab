package omfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type OmfsRule struct{}

func init() {
	rules.Register(&OmfsRule{})
}

func (r *OmfsRule) Name() string {
	return "omfs"
}

func (r *OmfsRule) Dependencies() []string {
	return []string{}
}

func (r *OmfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "omfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "omfs", omfsOptions), nil
}

var omfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"uid":   options.UIDOption{},
	"gid":   options.GIDOption{},
	"umask": options.UmaskOption{},
	"fmask": options.UmaskOption{},
	"dmask": options.UmaskOption{},
})
