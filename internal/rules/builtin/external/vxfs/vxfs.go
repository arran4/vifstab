package vxfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/external"
)

type VxFSRule struct{}

func init() {
	rules.Register(&VxFSRule{})
}

func (r *VxFSRule) Name() string {
	return "vxfs"
}

func (r *VxFSRule) Dependencies() []string {
	return []string{}
}

func (r *VxFSRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "vxfs" {
		return nil, nil
	}

	issues := external.CheckModuleSupport(ctx, fsType, entry, "vxfs", "")
	return issues, nil
}
