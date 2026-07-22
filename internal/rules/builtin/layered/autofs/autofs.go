package autofs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type AutofsRule struct{}

func init() {
	rules.Register(&AutofsRule{})
}

func (r *AutofsRule) Name() string {
	return "autofs"
}

func (r *AutofsRule) Dependencies() []string {
	return []string{}
}

func (r *AutofsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "autofs" {
		return nil, nil
	}

	return nil, nil
}
