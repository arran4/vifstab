package ncpfs

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

type NcpfsRule struct{}

func init() {
	rules.Register(&NcpfsRule{})
}

func (r *NcpfsRule) Name() string {
	return "ncpfs"
}

func (r *NcpfsRule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *NcpfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	return legacy.CheckFSVersion(ctx, entry, "ncpfs", kernel.Ver{}, kernel.Ver{}, false, "", false), nil
}
