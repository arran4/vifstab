package cgroup

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type CgroupRule struct{}

func init() {
	rules.Register(&CgroupRule{})
}

func (r *CgroupRule) Name() string {
	return "cgroup"
}

func (r *CgroupRule) Dependencies() []string {
	return []string{}
}

func (r *CgroupRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "cgroup" {
		return nil, nil
	}

	return nil, nil
}
