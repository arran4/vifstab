package ntfs3

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Ntfs3Rule struct{}

func init() {
	rules.Register(&Ntfs3Rule{})
}

func (r *Ntfs3Rule) Name() string {
	return "ntfs3"
}

func (r *Ntfs3Rule) Dependencies() []string {
	return []string{}
}

func (r *Ntfs3Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ntfs3" {
		return nil, nil
	}

	return rules.ValidateOptionsMap(ctx, api, entry, "ntfs3", ntfs3Options), nil
}

var ntfs3Options = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"iocharset":        rules.StringOption{},
	"uid":              rules.UIDOption{},
	"gid":              rules.GIDOption{},
	"umask":            rules.OctalOption{},
	"dmask":            rules.OctalOption{},
	"fmask":            rules.OctalOption{},
	"nohidden":         rules.NoValueOption{},
	"sys_immutable":    rules.NoValueOption{},
	"hide_dot_files":   rules.NoValueOption{},
	"windows_names":    rules.NoValueOption{},
	"discard":          rules.NoValueOption{},
	"force":            rules.NoValueOption{},
	"sparse":           rules.NoValueOption{},
	"showmeta":         rules.NoValueOption{},
	"prealloc":         rules.NoValueOption{},
	"acl":              rules.NoValueOption{},
	"hidden":           rules.NoValueOption{},
	"nosys_immutable":  rules.NoValueOption{},
	"nohide_dot_files": rules.NoValueOption{},
	"nowindows_names":  rules.NoValueOption{},
	"nodiscard":        rules.NoValueOption{},
	"noforce":          rules.NoValueOption{},
	"nosparse":         rules.NoValueOption{},
	"noshowmeta":       rules.NoValueOption{},
	"noprealloc":       rules.NoValueOption{},
	"noacl":            rules.NoValueOption{},
})
