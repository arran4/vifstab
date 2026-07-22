package mergerfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type MergerfsRule struct{}

func init() {
	rules.Register(&MergerfsRule{})
}

func (r *MergerfsRule) Name() string {
	return "mergerfs"
}

func (r *MergerfsRule) Dependencies() []string {
	return []string{}
}

func (r *MergerfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "mergerfs" {
		return nil, nil
	}

	return nil, nil
}
