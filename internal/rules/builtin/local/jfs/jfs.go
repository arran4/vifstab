package jfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

type JfsRule struct{}

func init() {
	rules.Register(&JfsRule{})
}

func (r *JfsRule) Name() string {
	return "jfs"
}

func (r *JfsRule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *JfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "jfs" {
		return nil, nil
	}

	issues := legacy.CheckFSVersion(ctx, entry, "jfs", kernel.Ver{}, kernel.Ver{}, false, "", true)

	return issues, nil
}
