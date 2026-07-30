package sysfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type SysfsRule struct{}

func init() {
	rules.Register(&SysfsRule{})
}

func (r *SysfsRule) Name() string {
	return "sysfs"
}

func (r *SysfsRule) Dependencies() []string {
	return []string{}
}

func (r *SysfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "sysfs" {
		return nil, nil
	}

	return rules.ValidateOptionsMap(ctx, api, entry, "sysfs", sysfsOptions), nil
}

var sysfsOptions = rules.CommonOptions()
