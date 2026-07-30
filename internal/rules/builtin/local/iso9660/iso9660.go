package iso9660

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
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

	return rules.ValidateOptionsMap(ctx, api, entry, "iso9660", iso9660Options), nil
}

var iso9660Options = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"norock":             rules.NoValueOption{},
	"rock":               rules.NoValueOption{},
	"nojoliet":           rules.NoValueOption{},
	"joliet":             rules.NoValueOption{},
	"check":              rules.EnumOption{Allowed: map[string]bool{"r": true, "relaxed": true, "s": true, "strict": true}},
	"uid":                rules.UIDOption{},
	"gid":                rules.GIDOption{},
	"map":                rules.EnumOption{Allowed: map[string]bool{"n": true, "normal": true, "o": true, "off": true, "a": true, "acorn": true}},
	"mode":               rules.OctalOption{},
	"dmode":              rules.OctalOption{},
	"umask":              rules.OctalOption{},
	"unhide":             rules.NoValueOption{},
	"nounhide":           rules.NoValueOption{},
	"block":              rules.EnumOption{Allowed: map[string]bool{"512": true, "1024": true, "2048": true}, CustomErrMsg: "Invalid value '%s' for iso9660 mount option 'block' on line %d, must be 512, 1024, or 2048"},
	"conv":               rules.StringOption{},
	"cruft":              rules.NoValueOption{},
	"nocruft":            rules.NoValueOption{},
	"session":            rules.IntOption{},
	"sbsector":           rules.IntOption{},
	"iocharset":          rules.StringOption{},
	"utf8":               rules.NoValueOption{},
	"noutf8":             rules.NoValueOption{},
	"overriderockperm":   rules.NoValueOption{},
	"nooverriderockperm": rules.NoValueOption{},
	"hide":               rules.NoValueOption{},
	"nohide":             rules.NoValueOption{},
	"showassoc":          rules.NoValueOption{},
	"noshowassoc":        rules.NoValueOption{},
})
