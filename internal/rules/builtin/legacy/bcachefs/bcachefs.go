package bcachefs

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

type BcachefsRule struct{}

func init() {
	rules.Register(&BcachefsRule{})
}

func (r *BcachefsRule) Name() string {
	return "bcachefs"
}

func (r *BcachefsRule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *BcachefsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	// bcachefs is not present in the current upstream filesystem tree (or wasn't for a long time)
	return legacy.CheckFSVersion(ctx, api, entry, "bcachefs", kernel.Ver{Major: 6, Minor: 7, Patch: 0}, kernel.Ver{Major: 6, Minor: 18, Patch: 0}, false, "", false), nil
}
