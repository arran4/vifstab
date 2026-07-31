package v9fs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

var v9fsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"trans":      options.EnumOption{Allowed: map[string]bool{"unix": true, "tcp": true, "fd": true, "virtio": true, "rdma": true, "usbg": true}},
	"uname":      options.StringOption{},
	"aname":      options.StringOption{},
	"cache":      options.StringOption{},
	"debug":      options.StringOption{}, // can be bitmask or number string
	"rfdno":      options.IntOption{},
	"wfdno":      options.IntOption{},
	"msize":      options.IntOption{},
	"port":       options.IntOption{},
	"noextend":   options.NoValueOption{},
	"extend":     options.NoValueOption{},
	"version":    options.EnumOption{Allowed: map[string]bool{"9p2000": true, "9p2000.u": true, "9p2000.L": true}},
	"dfltuid":    options.OptionalUIDOption{}, // 9p can also just be `uname=something` without giving dfltuid
	"dfltgid":    options.OptionalGIDOption{},
	"afid":       options.StringOption{}, // security channel
	"nodevmap":   options.NoValueOption{},
	"devmap":     options.NoValueOption{},
	"directio":   options.NoValueOption{},
	"nodirectio": options.NoValueOption{},
	"ignoreqv":   options.NoValueOption{},
	"noignoreqv": options.NoValueOption{},
	"noxattr":    options.NoValueOption{},
	"xattr":      options.NoValueOption{},
	"access":     options.UIDOrEnumOption{AllowedEnum: map[string]bool{"user": true, "any": true, "client": true}},
	"cachetag":   options.StringOption{},
	"negtimeout": options.IntOption{},
})

type V9fsRule struct{}

func init() {
	rules.Register(&V9fsRule{})
}

func (r *V9fsRule) Name() string {
	return "9p"
}

func (r *V9fsRule) Dependencies() []string {
	return []string{}
}

func (r *V9fsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "9p" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "9p", v9fsOptions), nil
}
