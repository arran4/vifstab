package adfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type AdfsRule struct{}

func init() {
	rules.Register(&AdfsRule{})
}

func (r *AdfsRule) Name() string {
	return "adfs"
}

func (r *AdfsRule) Dependencies() []string {
	return []string{}
}

func (r *AdfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "adfs" {
		return nil, nil
	}

	return nil, nil
}
