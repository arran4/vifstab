package sysv

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

type SysvRule struct{}

func init() {
	rules.Register(&SysvRule{})
}

func (r *SysvRule) Name() string {
	return "sysv"
}

func (r *SysvRule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *SysvRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	return legacy.CheckFSVersion(ctx, api, entry, "sysv", kernel.Ver{}, kernel.Ver{}, false, "", false), nil
}
