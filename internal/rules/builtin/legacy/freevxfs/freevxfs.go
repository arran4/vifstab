package freevxfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type FreevxfsRule struct{}

func init() {
	rules.Register(&FreevxfsRule{})
}

func (r *FreevxfsRule) Name() string {
	return "freevxfs"
}

func (r *FreevxfsRule) Dependencies() []string {
	return []string{}
}

func (r *FreevxfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "freevxfs" {
		return nil, nil
	}

	return nil, nil
}
