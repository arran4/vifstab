package ntfs3

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
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

	return options.ValidateOptionsMap(ctx, api, entry, "ntfs3", ntfs3Options), nil
}

var ntfs3Options = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"iocharset": options.IOCharsetOption{},
	"uid":              options.UIDOption{},
	"gid":              options.GIDOption{},
	"umask":            options.OctalOption{},
	"dmask":            options.OctalOption{},
	"fmask":            options.OctalOption{},
	"nohidden":         options.NoValueOption{},
	"sys_immutable":    options.NoValueOption{},
	"hide_dot_files":   options.NoValueOption{},
	"windows_names":    options.NoValueOption{},
	"discard":          options.NoValueOption{},
	"force":            options.NoValueOption{},
	"sparse":           options.NoValueOption{},
	"showmeta":         options.NoValueOption{},
	"prealloc":         options.NoValueOption{},
	"acl":              options.NoValueOption{},
	"hidden":           options.NoValueOption{},
	"nosys_immutable":  options.NoValueOption{},
	"nohide_dot_files": options.NoValueOption{},
	"nowindows_names":  options.NoValueOption{},
	"nodiscard":        options.NoValueOption{},
	"noforce":          options.NoValueOption{},
	"nosparse":         options.NoValueOption{},
	"noshowmeta":       options.NoValueOption{},
	"noprealloc":       options.NoValueOption{},
	"noacl":            options.NoValueOption{},
})
