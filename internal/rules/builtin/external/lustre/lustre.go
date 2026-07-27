package lustre

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/external"
)

type LustreRule struct{}

func init() {
	rules.Register(&LustreRule{})
}

func (r *LustreRule) Name() string {
	return "lustre"
}

func (r *LustreRule) Dependencies() []string {
	return []string{}
}

func (r *LustreRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "lustre" {
		return nil, nil
	}

	issues := external.CheckModuleSupport(ctx, fsType, entry, "lustre", "")
	return issues, nil
}
