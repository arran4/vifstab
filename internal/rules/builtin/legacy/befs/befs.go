package befs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type BefsRule struct{}

func init() {
	rules.Register(&BefsRule{})
}

func (r *BefsRule) Name() string {
	return "befs"
}

func (r *BefsRule) Dependencies() []string {
	return []string{}
}

func (r *BefsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "befs" {
		return nil, nil
	}

	return nil, nil
}
