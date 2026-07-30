package ext2

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Ext2Rule struct{}

func init() {
	rules.Register(&Ext2Rule{})
}

func (r *Ext2Rule) Name() string {
	return "ext2"
}

func (r *Ext2Rule) Dependencies() []string {
	return []string{}
}

func (r *Ext2Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ext2" {
		return nil, nil
	}

	return rules.ValidateOptionsMap(ctx, api, entry, "ext2", ext2Options), nil
}

var ext2Options = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"acl":          rules.NoValueOption{},
	"noacl":        rules.NoValueOption{},
	"bsddf":        rules.NoValueOption{},
	"minixdf":      rules.NoValueOption{},
	"check":        rules.EnumOption{Allowed: map[string]bool{"none": true, "nocheck": true}, CustomErrMsg: "Unsupported value '%s' for ext2 mount option 'check' on line %d (only 'none' or 'nocheck' are supported)"},
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
})
