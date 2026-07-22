package xfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type XfsRule struct{}

func init() {
	rules.Register(&XfsRule{})
}

func (r *XfsRule) Name() string {
	return "xfs"
}

func (r *XfsRule) Dependencies() []string {
	return []string{}
}

func (r *XfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "xfs" {
		return nil, nil
	}

	return nil, nil
}
