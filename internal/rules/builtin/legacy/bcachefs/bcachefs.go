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
	return []string{}
}

func (r *BcachefsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	// bcachefs is not present in the current upstream filesystem tree (or wasn't for a long time)
	return legacy.CheckFSVersion(ctx, entry, "bcachefs", kernel.Ver{}, kernel.Ver{}, false, "", false), nil
}
