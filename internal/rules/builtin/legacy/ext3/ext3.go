package ext3

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

type Ext3Rule struct{}

func init() {
	rules.Register(&Ext3Rule{})
}

func (r *Ext3Rule) Name() string {
	return "ext3"
}

func (r *Ext3Rule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *Ext3Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	// ext3 was removed in 4.3, replaced by ext4 handling ext3 format
	return legacy.CheckFSVersion(ctx, entry, "ext3", kernel.Ver{}, kernel.Ver{Major: 4, Minor: 3, Patch: 0}, false, "ext4 handles ext3-format volumes", false), nil
}
