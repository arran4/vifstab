package vfat

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/local/fat"
	"github.com/arran4/vifstab/internal/rules/options"
)

type VfatRule struct{}

func init() {
	rules.Register(&VfatRule{})
}

func (r *VfatRule) Name() string {
	return "vfat"
}

func (r *VfatRule) Dependencies() []string {
	return []string{}
}

func (r *VfatRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "vfat" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "vfat", vfatOptions), nil
}

var vfatOptions = options.MergeOptions(fat.Options, map[string]options.OptionValidator{
	"dotsOK":    options.IncorrectOption{Reason: "vfat explicitly kills dotsOK mount option"},
	"uni_xlate": options.BoolOption{Sets: options.BoolSetStandard},
	"posix":     options.BoolOption{Sets: options.BoolSetStandard},
	"nonumtail": options.BoolOption{Sets: options.BoolSetStandard},
	"utf8":      options.BoolOption{Sets: options.BoolSetStandard},
	"shortname": options.EnumOption{
		Allowed: map[string]bool{"lower": true, "win95": true, "winnt": true, "mixed": true},
	},
})
