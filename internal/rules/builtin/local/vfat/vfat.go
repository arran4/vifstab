package vfat

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type VfatRule struct{}

func init() {
	rules.Register(&VfatRule{})
}

func (r *VfatRule) Name() string {
	return "vfat"
}

func (r *VfatRule) Dependencies() []string {
	return []string{}
}

func (r *VfatRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "vfat" {
		return nil, nil
	}

	return nil, nil
}
