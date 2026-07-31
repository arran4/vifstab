package ceph

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type CephRule struct{}

func init() {
	rules.Register(&CephRule{})
}

func (r *CephRule) Name() string {
	return "ceph"
}

func (r *CephRule) Dependencies() []string {
	return []string{}
}

func (r *CephRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ceph" {
		return nil, nil
	}

	fsOptions := options.MergeOptions(options.CommonOptions(), cephOptions)
	issues := options.ValidateOptionsMap(ctx, api, entry, "ceph", fsOptions)

	return issues, nil
}

var cephOptions = map[string]options.OptionValidator{
	// Basic options
	"name":            options.StringOption{},
	"conf":            options.StringOption{},
	"mount_timeout":   options.IntOption{},
	"ms_mode":         options.EnumOption{Allowed: map[string]bool{"legacy": true, "crc": true, "secure": true, "prefer-crc": true, "prefer-secure": true}},
	"mon_addr":        options.StringOption{},
	"fsid":            options.StringOption{},
	"secret":          options.StringOption{},
	"secretfile":      options.StringOption{},
	"recover_session": options.EnumOption{Allowed: map[string]bool{"no": true, "clean": true}},
	"fs":              options.StringOption{},
	"mds_namespace":   options.StringOption{},

	// Advanced options
	"cap_release_safety":    options.IntOption{},
	"caps_wanted_delay_max": options.IntOption{},
	"caps_wanted_delay_min": options.IntOption{},
	"dirstat":               options.NoValueOption{},
	"nodirstat":             options.NoValueOption{},
	"ip":                    options.StringOption{},
	"noasyncreaddir":        options.NoValueOption{},
	"nocrc":                 options.NoValueOption{},
	"noshare":               options.NoValueOption{},
	"osdkeepalive":          options.IntOption{},
	"osd_idle_ttl":          options.IntOption{},
	"rasize":                options.IntOption{},
	"rbytes":                options.NoValueOption{},
	"norbytes":              options.NoValueOption{},
	"readdir_max_bytes":     options.IntOption{},
	"readdir_max_entries":   options.IntOption{},
	"rsize":                 options.IntOption{},
	"snapdirname":           options.StringOption{},
	"write_congestion_kb":   options.IntOption{},
	"wsize":                 options.IntOption{},
	"wsync":                 options.NoValueOption{},
	"nowsync":               options.NoValueOption{},
	"crush_location":        options.StringOption{},
	"read_from_replica":     options.EnumOption{Allowed: map[string]bool{"no": true, "balance": true, "localize": true}},
}
