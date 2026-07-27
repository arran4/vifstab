package selinuxfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type SelinuxfsRule struct{}

func init() {
	rules.Register(&SelinuxfsRule{})
}

func (r *SelinuxfsRule) Name() string {
	return "selinuxfs"
}

func (r *SelinuxfsRule) Dependencies() []string {
	return []string{}
}

func (r *SelinuxfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "selinuxfs" {
		return nil, nil
	}

	return nil, nil
}
