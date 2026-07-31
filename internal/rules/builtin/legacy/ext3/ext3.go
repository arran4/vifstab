package ext3

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

type Ext3Rule struct{}

func init() {
	rules.Register(&Ext3Rule{})
}

func (r *Ext3Rule) Name() string {
	return "ext3"
}

func (r *Ext3Rule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *Ext3Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	// ext3 was removed in 4.3, replaced by ext4 handling ext3 format
	issues := legacy.CheckFSVersion(ctx, api, entry, "ext3", kernel.Ver{}, kernel.Ver{}, false, "ext4 handles ext3-format volumes", false)

	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return issues, nil
	}
	if len(entry.Fields) < 4 {
		return issues, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ext3" {
		return issues, nil
	}

	optsIssues := options.ValidateOptionsMap(ctx, api, entry, "ext3", ext3Options)
	return append(issues, optsIssues...), nil
}

var ext3Options = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"acl":          options.NoValueOption{},
	"noacl":        options.NoValueOption{},
	"bsddf":        options.NoValueOption{},
	"minixdf":      options.NoValueOption{},
	"check":        options.EnumOption{Allowed: map[string]bool{"none": true, "nocheck": true}, CustomErrMsg: "Unsupported value '%s' for ext3 mount option 'check' on line %d (only 'none' or 'nocheck' are supported)"},
	"nocheck":      options.NoValueOption{},
	"debug":        options.NoValueOption{},
	"errors":       options.EnumOption{Allowed: map[string]bool{"continue": true, "remount-ro": true, "panic": true}},
	"grpid":        options.NoValueOption{},
	"bsdgroups":    options.NoValueOption{},
	"nogrpid":      options.NoValueOption{},
	"sysvgroups":   options.NoValueOption{},
	"grpquota":     options.NoValueOption{},
	"usrquota":     options.NoValueOption{},
	"quota":        options.NoValueOption{},
	"noquota":      options.NoValueOption{},
	"nouid32":      options.NoValueOption{},
	"oldalloc":     options.NoValueOption{},
	"orlov":        options.NoValueOption{},
	"resgid":       options.IntOption{},
	"resuid":       options.IntOption{},
	"sb":           options.IntOption{},
	"user_xattr":   options.NoValueOption{},
	"nouser_xattr": options.NoValueOption{},

	"journal_dev":  options.IntOption{},
	"journal_path": options.TODOOption{Comment: "validate journal device path"},
	"norecovery":   options.NoValueOption{},
	"noload":       options.NoValueOption{},
	"data":         options.EnumOption{Allowed: map[string]bool{"journal": true, "ordered": true, "writeback": true}},
	"data_err":     options.EnumOption{Allowed: map[string]bool{"abort": true, "ignore": true}},
	"barrier":      options.EnumOption{Allowed: map[string]bool{"0": true, "1": true}},
	"commit":       options.IntOption{},
	"jqfmt":        options.EnumOption{Allowed: map[string]bool{"vfsold": true, "vfsv0": true, "vfsv1": true}},
	"usrjquota":    options.TODOOption{Comment: "validate user quota path"},
	"grpjquota":    options.TODOOption{Comment: "validate group quota path"},
})
