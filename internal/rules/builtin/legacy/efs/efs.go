package efs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type EfsRule struct{}

func init() {
	rules.Register(&EfsRule{})
}

func (r *EfsRule) Name() string {
	return "efs"
}

func (r *EfsRule) Dependencies() []string {
	return []string{}
}

func (r *EfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "efs" {
		return nil, nil
	}

	return nil, nil
}
