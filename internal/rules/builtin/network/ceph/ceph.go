package ceph

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
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

func (r *CephRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
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

	return nil, nil
}
