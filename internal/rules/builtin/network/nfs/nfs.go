package nfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type NfsRule struct{}

func init() {
	rules.Register(&NfsRule{})
}

func (r *NfsRule) Name() string {
	return "nfs"
}

func (r *NfsRule) Dependencies() []string {
	return []string{}
}

func (r *NfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "nfs" {
		return nil, nil
	}

	return nil, nil
}
