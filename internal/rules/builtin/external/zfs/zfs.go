package zfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/external"
)

type ZFSRule struct{}

func init() {
	rules.Register(&ZFSRule{})
}

func (r *ZFSRule) Name() string {
	return "zfs"
}

func (r *ZFSRule) Dependencies() []string {
	return []string{}
}

func (r *ZFSRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "zfs" {
		return nil, nil
	}

	issues := external.CheckModuleSupport(ctx, fsType, entry, "zfs", "")
	return issues, nil
}
