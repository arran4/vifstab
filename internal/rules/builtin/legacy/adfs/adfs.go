package adfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type AdfsRule struct{}

func init() {
	rules.Register(&AdfsRule{})
}

func (r *AdfsRule) Name() string {
	return "adfs"
}

func (r *AdfsRule) Dependencies() []string {
	return []string{}
}

func (r *AdfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "adfs" {
		return nil, nil
	}

	return rules.ValidateOptionsMap(ctx, api, entry, "adfs", adfsOptions), nil
}

var adfsOptions = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"uid":     rules.UIDOption{},
	"gid":     rules.GIDOption{},
	"ownmask": rules.OctalOption{},
	"othmask": rules.OctalOption{},
})
