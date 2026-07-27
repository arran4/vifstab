package fuse

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type FuseRule struct{}

func init() {
	rules.Register(&FuseRule{})
}

func (r *FuseRule) Name() string {
	return "fuse"
}

func (r *FuseRule) Dependencies() []string {
	return []string{}
}

func (r *FuseRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "fuse" {
		return nil, nil
	}

	return nil, nil
}
