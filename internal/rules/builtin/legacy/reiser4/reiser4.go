package reiser4

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

type Reiser4Rule struct{}

func init() {
	rules.Register(&Reiser4Rule{})
}

func (r *Reiser4Rule) Name() string {
	return "reiser4"
}

func (r *Reiser4Rule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *Reiser4Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	return legacy.CheckFSVersion(ctx, entry, "reiser4", kernel.Ver{}, kernel.Ver{}, true, "", false), nil
}
