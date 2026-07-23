package reiser5

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

type Reiser5Rule struct{}

func init() {
	rules.Register(&Reiser5Rule{})
}

func (r *Reiser5Rule) Name() string {
	return "reiser5"
}

func (r *Reiser5Rule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *Reiser5Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	return legacy.CheckFSVersion(ctx, entry, "reiser5", kernel.Ver{}, kernel.Ver{}, true, "", false), nil
}
