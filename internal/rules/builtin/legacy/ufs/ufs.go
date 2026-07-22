package ufs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type UfsRule struct{}

func init() {
	rules.Register(&UfsRule{})
}

func (r *UfsRule) Name() string {
	return "ufs"
}

func (r *UfsRule) Dependencies() []string {
	return []string{}
}

func (r *UfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ufs" {
		return nil, nil
	}

	return nil, nil
}
