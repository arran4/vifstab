package iso9660

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type Iso9660Rule struct{}

func init() {
	rules.Register(&Iso9660Rule{})
}

func (r *Iso9660Rule) Name() string {
	return "iso9660"
}

func (r *Iso9660Rule) Dependencies() []string {
	return []string{}
}

func (r *Iso9660Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "iso9660" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "iso9660", iso9660Options), nil
}

var iso9660Options = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"norock":             options.NoValueOption{},
	"rock":               options.NoValueOption{},
	"nojoliet":           options.NoValueOption{},
	"joliet":             options.NoValueOption{},
	"check":              options.EnumOption{Allowed: map[string]bool{"r": true, "relaxed": true, "s": true, "strict": true}},
	"uid":                options.UIDOption{},
	"gid":                options.GIDOption{},
	"map":                options.EnumOption{Allowed: map[string]bool{"n": true, "normal": true, "o": true, "off": true, "a": true, "acorn": true}},
	"mode":               options.OctalOption{},
	"dmode":              options.OctalOption{},
	"umask":              options.OctalOption{},
	"unhide":             options.NoValueOption{},
	"nounhide":           options.NoValueOption{},
	"block":              options.EnumOption{Allowed: map[string]bool{"512": true, "1024": true, "2048": true}, CustomErrMsg: "Invalid value '%s' for iso9660 mount option 'block' on line %d, must be 512, 1024, or 2048"},
	"conv":               options.StringOption{},
	"cruft":              options.NoValueOption{},
	"nocruft":            options.NoValueOption{},
	"session":            options.IntOption{},
	"sbsector":           options.IntOption{},
	"iocharset":          options.IOCharsetOption{},
	"utf8":               options.NoValueOption{},
	"noutf8":             options.NoValueOption{},
	"overriderockperm":   options.NoValueOption{},
	"nooverriderockperm": options.NoValueOption{},
	"hide":               options.NoValueOption{},
	"nohide":             options.NoValueOption{},
	"showassoc":          options.NoValueOption{},
	"noshowassoc":        options.NoValueOption{},
})
