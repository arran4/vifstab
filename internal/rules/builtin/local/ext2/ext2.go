package ext2

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
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

	return options.ValidateOptionsMap(ctx, api, entry, "ext2", ext2Options), nil
}

var ext2Options = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"acl":          options.NoValueOption{},
	"noacl":        options.NoValueOption{},
	"bsddf":        options.NoValueOption{},
	"minixdf":      options.NoValueOption{},
	"check":        options.EnumOption{Allowed: map[string]bool{"none": true, "nocheck": true}, CustomErrMsg: "Unsupported value '%s' for ext2 mount option 'check' on line %d (only 'none' or 'nocheck' are supported)"},
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
})
