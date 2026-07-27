package fat

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type FatRule struct{}

func init() {
	rules.Register(&FatRule{})
}

func (r *FatRule) Name() string {
	return "fat"
}

func (r *FatRule) Dependencies() []string {
	return []string{}
}

func (r *FatRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "fat" {
		return nil, nil
	}

	return nil, nil
}
