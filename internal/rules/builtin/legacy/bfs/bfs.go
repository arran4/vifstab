package bfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type BfsRule struct{}

func init() {
	rules.Register(&BfsRule{})
}

func (r *BfsRule) Name() string {
	return "bfs"
}

func (r *BfsRule) Dependencies() []string {
	return []string{}
}

func (r *BfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "bfs" {
		return nil, nil
	}

	return nil, nil
}
