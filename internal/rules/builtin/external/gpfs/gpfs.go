package gpfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/external"
)

type GPFSRule struct{}

func init() {
	rules.Register(&GPFSRule{})
}

func (r *GPFSRule) Name() string {
	return "gpfs"
}

func (r *GPFSRule) Dependencies() []string {
	return []string{}
}

func (r *GPFSRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "gpfs" {
		return nil, nil
	}

	issues := external.CheckModuleSupport(ctx, fsType, entry, "mmfs26", "")
	return issues, nil
}
