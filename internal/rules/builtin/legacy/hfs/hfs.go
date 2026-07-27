package hfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type HfsRule struct{}

func init() {
	rules.Register(&HfsRule{})
}

func (r *HfsRule) Name() string {
	return "hfs"
}

func (r *HfsRule) Dependencies() []string {
	return []string{}
}

func (r *HfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "hfs" {
		return nil, nil
	}

	return nil, nil
}
