package beegfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/external"
)

type BeeGFSRule struct{}

func init() {
	rules.Register(&BeeGFSRule{})
}

func (r *BeeGFSRule) Name() string {
	return "beegfs"
}

func (r *BeeGFSRule) Dependencies() []string {
	return []string{}
}

func (r *BeeGFSRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "beegfs" {
		return nil, nil
	}

	issues := external.CheckModuleSupport(ctx, fsType, entry, "beegfs", "")
	return issues, nil
}
