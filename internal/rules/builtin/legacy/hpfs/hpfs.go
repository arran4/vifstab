package hpfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type HpfsRule struct{}

func init() {
	rules.Register(&HpfsRule{})
}

func (r *HpfsRule) Name() string {
	return "hpfs"
}

func (r *HpfsRule) Dependencies() []string {
	return []string{}
}

func (r *HpfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "hpfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "hpfs", hpfsOptions), nil
}

var hpfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"uid":       options.UIDOption{},
	"gid":       options.GIDOption{},
	"umask":     options.UmaskOption{},
	"case":      options.EnumOption{Allowed: map[string]bool{"lower": true, "asis": true}},
	"conv":      options.EnumOption{Allowed: map[string]bool{"binary": true, "text": true, "auto": true}},
	"check":     options.EnumOption{Allowed: map[string]bool{"none": true, "normal": true, "strict": true}},
	"errors":    options.EnumOption{Allowed: map[string]bool{"continue": true, "remount-ro": true, "panic": true}},
	"chkdsk":    options.EnumOption{Allowed: map[string]bool{"no": true, "errors": true, "always": true}},
	"eas":       options.EnumOption{Allowed: map[string]bool{"no": true, "ro": true, "rw": true}},
	"timeshift": options.IntOption{},
})
