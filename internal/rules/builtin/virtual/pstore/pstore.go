package pstore

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type PstoreRule struct{}

func init() {
	rules.Register(&PstoreRule{})
}

func (r *PstoreRule) Name() string {
	return "pstore"
}

func (r *PstoreRule) Dependencies() []string {
	return []string{}
}

func (r *PstoreRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "pstore" {
		return nil, nil
	}

	return nil, nil
}
