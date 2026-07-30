package usbfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
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

	return options.ValidateOptionsMap(ctx, api, entry, "usbfs", usbfsOptions), nil
}

var usbfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"devuid":   options.UIDOption{},
	"busuid":   options.UIDOption{},
	"listuid":  options.UIDOption{},
	"devgid":   options.GIDOption{},
	"busgid":   options.GIDOption{},
	"listgid":  options.GIDOption{},
	"devmode":  options.OctalOption{},
	"busmode":  options.OctalOption{},
	"listmode": options.OctalOption{},
})
