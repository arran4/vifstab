package ufs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type UfsRule struct{}

func init() {
	rules.Register(&UfsRule{})
}

func (r *UfsRule) Name() string {
	return "ufs"
}

func (r *UfsRule) Dependencies() []string {
	return []string{}
}

func (r *UfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ufs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "ufs", ufsOptions), nil
}

var ufsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"ufstype": options.EnumOption{Allowed: map[string]bool{
		"old": true, "44bsd": true, "ufs2": true, "5xbsd": true, "sun": true, "sunx86": true, "hp": true, "nextstep": true, "nextstep-cd": true, "openstep": true,
	}},
	"onerror": options.EnumOption{Allowed: map[string]bool{
		"panic": true, "lock": true, "umount": true, "repair": true,
	}},
})
