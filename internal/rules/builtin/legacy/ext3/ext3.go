package ext3

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
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

	optsIssues := rules.ValidateOptionsMap(ctx, api, entry, "ext3", ext3Options)
	return append(issues, optsIssues...), nil
}

var ext3Options = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"acl":          rules.NoValueOption{},
	"noacl":        rules.NoValueOption{},
	"bsddf":        rules.NoValueOption{},
	"minixdf":      rules.NoValueOption{},
	"check":        rules.EnumOption{Allowed: map[string]bool{"none": true, "nocheck": true}, CustomErrMsg: "Unsupported value '%s' for ext3 mount option 'check' on line %d (only 'none' or 'nocheck' are supported)"},
	"nocheck":      rules.NoValueOption{},
	"debug":        rules.NoValueOption{},
	"errors":       rules.EnumOption{Allowed: map[string]bool{"continue": true, "remount-ro": true, "panic": true}},
	"grpid":        rules.NoValueOption{},
	"bsdgroups":    rules.NoValueOption{},
	"nogrpid":      rules.NoValueOption{},
	"sysvgroups":   rules.NoValueOption{},
	"grpquota":     rules.NoValueOption{},
	"usrquota":     rules.NoValueOption{},
	"quota":        rules.NoValueOption{},
	"noquota":      rules.NoValueOption{},
	"nouid32":      rules.NoValueOption{},
	"oldalloc":     rules.NoValueOption{},
	"orlov":        rules.NoValueOption{},
	"resgid":       rules.IntOption{},
	"resuid":       rules.IntOption{},
	"sb":           rules.IntOption{},
	"user_xattr":   rules.NoValueOption{},
	"nouser_xattr": rules.NoValueOption{},

	"journal_dev":  rules.IntOption{},
	"journal_path": rules.StringOption{},
	"norecovery":   rules.NoValueOption{},
	"noload":       rules.NoValueOption{},
	"data":         rules.EnumOption{Allowed: map[string]bool{"journal": true, "ordered": true, "writeback": true}},
	"data_err":     rules.EnumOption{Allowed: map[string]bool{"abort": true, "ignore": true}},
	"barrier":      rules.EnumOption{Allowed: map[string]bool{"0": true, "1": true}},
	"commit":       rules.IntOption{},
	"jqfmt":        rules.EnumOption{Allowed: map[string]bool{"vfsold": true, "vfsv0": true, "vfsv1": true}},
	"usrjquota":    rules.StringOption{},
	"grpjquota":    rules.StringOption{},
})
