package vboxsf

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type VboxsfRule struct{}

func init() {
	rules.Register(&VboxsfRule{})
}

func (r *VboxsfRule) Name() string {
	return "vboxsf"
}

func (r *VboxsfRule) Dependencies() []string {
	return []string{}
}

func (r *VboxsfRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "vboxsf" {
		return nil, nil
	}

	return nil, nil
}
