package usbfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type UsbfsRule struct{}

func init() {
	rules.Register(&UsbfsRule{})
}

func (r *UsbfsRule) Name() string {
	return "usbfs"
}

func (r *UsbfsRule) Dependencies() []string {
	return []string{}
}

func (r *UsbfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "usbfs" {
		return nil, nil
	}

	return rules.ValidateOptionsMap(ctx, api, entry, "usbfs", usbfsOptions), nil
}

var usbfsOptions = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"devuid":   rules.UIDOption{},
	"busuid":   rules.UIDOption{},
	"listuid":  rules.UIDOption{},
	"devgid":   rules.GIDOption{},
	"busgid":   rules.GIDOption{},
	"listgid":  rules.GIDOption{},
	"devmode":  rules.OctalOption{},
	"busmode":  rules.OctalOption{},
	"listmode": rules.OctalOption{},
})
