package openafs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/external"
)

type OpenAFSRule struct{}

func init() {
	rules.Register(&OpenAFSRule{})
}

func (r *OpenAFSRule) Name() string {
	return "openafs"
}

func (r *OpenAFSRule) Dependencies() []string {
	return []string{}
}

func (r *OpenAFSRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "openafs" {
		return nil, nil
	}

	issues := external.CheckModuleSupport(ctx, fsType, entry, "openafs", "")
	return issues, nil
}
