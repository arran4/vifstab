package apfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type ApfsRule struct{}

func init() {
	rules.Register(&ApfsRule{})
}

func (r *ApfsRule) Name() string {
	return "apfs"
}

func (r *ApfsRule) Dependencies() []string {
	return []string{}
}

func (r *ApfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "apfs" {
		return nil, nil
	}

	return nil, nil
}
