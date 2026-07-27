package cuse

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type CuseRule struct{}

func init() {
	rules.Register(&CuseRule{})
}

func (r *CuseRule) Name() string {
	return "cuse"
}

func (r *CuseRule) Dependencies() []string {
	return []string{}
}

func (r *CuseRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "cuse" {
		return nil, nil
	}

	return nil, nil
}
