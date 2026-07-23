package reiserfs

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

type ReiserfsRule struct{}

func init() {
	rules.Register(&ReiserfsRule{})
}

func (r *ReiserfsRule) Name() string {
	return "reiserfs"
}

func (r *ReiserfsRule) Dependencies() []string {
	return []string{}
}

func (r *ReiserfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	// ReiserFS was removed in 6.6
	return legacy.CheckFSVersion(ctx, entry, "reiserfs", kernel.Ver{}, kernel.Ver{Major: 6, Minor: 6, Patch: 0}, false, "", false), nil
}
