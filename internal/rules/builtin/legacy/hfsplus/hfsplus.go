package hfsplus

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type HfsPlusRule struct{}

func init() {
	rules.Register(&HfsPlusRule{})
}

func (r *HfsPlusRule) Name() string {
	return "hfsplus"
}

func (r *HfsPlusRule) Dependencies() []string {
	return []string{}
}

func (r *HfsPlusRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "hfsplus" {
		return nil, nil
	}

	return nil, nil
}
