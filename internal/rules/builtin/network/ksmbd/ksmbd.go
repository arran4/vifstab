package ksmbd

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type KsmbdRule struct{}

func init() {
	rules.Register(&KsmbdRule{})
}

func (r *KsmbdRule) Name() string {
	return "ksmbd"
}

func (r *KsmbdRule) Dependencies() []string {
	return []string{}
}

func (r *KsmbdRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ksmbd" {
		return nil, nil
	}

	return nil, nil
}
