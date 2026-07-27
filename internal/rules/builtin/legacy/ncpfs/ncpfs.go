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

func (r *NcpfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	return legacy.CheckFSVersion(ctx, api, entry, "ncpfs", kernel.Ver{}, kernel.Ver{Major: 4, Minor: 17, Patch: 0}, false, "", false), nil
}
