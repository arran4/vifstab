package ubifs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type UbifsRule struct{}

func init() {
	rules.Register(&UbifsRule{})
}

func (r *UbifsRule) Name() string {
	return "ubifs"
}

func (r *UbifsRule) Dependencies() []string {
	return []string{}
}

func (r *UbifsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ubifs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "ubifs", ubifsOptions), nil
}

var ubifsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"bulk_read":       options.NoValueOption{},
	"no_bulk_read":    options.NoValueOption{},
	"chk_data_crc":    options.NoValueOption{},
	"no_chk_data_crc": options.NoValueOption{},
	"compr":           options.EnumOption{Allowed: map[string]bool{"none": true, "lzo": true, "zlib": true}},
	"auth_key":        options.StringOption{},
	"auth_hash_name":  options.StringOption{},
})
