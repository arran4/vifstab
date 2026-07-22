package fusectl

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type FusectlRule struct{}

func init() {
	rules.Register(&FusectlRule{})
}

func (r *FusectlRule) Name() string {
	return "fusectl"
}

func (r *FusectlRule) Dependencies() []string {
	return []string{}
}

func (r *FusectlRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "fusectl" {
		return nil, nil
	}

	return nil, nil
}
