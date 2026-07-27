package securityfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type SecurityfsRule struct{}

func init() {
	rules.Register(&SecurityfsRule{})
}

func (r *SecurityfsRule) Name() string {
	return "securityfs"
}

func (r *SecurityfsRule) Dependencies() []string {
	return []string{}
}

func (r *SecurityfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "securityfs" {
		return nil, nil
	}

	return nil, nil
}
