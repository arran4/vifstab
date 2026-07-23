package jffs

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

type JffsRule struct{}

func init() {
	rules.Register(&JffsRule{})
}

func (r *JffsRule) Name() string {
	return "jffs"
}

func (r *JffsRule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *JffsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	return legacy.CheckFSVersion(ctx, entry, "jffs", kernel.Ver{}, kernel.Ver{Major: 2, Minor: 6, Patch: 17}, false, "", false), nil
}
