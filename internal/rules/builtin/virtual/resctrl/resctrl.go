package resctrl

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type ResctrlRule struct{}

func init() {
	rules.Register(&ResctrlRule{})
}

func (r *ResctrlRule) Name() string {
	return "resctrl"
}

func (r *ResctrlRule) Dependencies() []string {
	return []string{}
}

func (r *ResctrlRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "resctrl" {
		return nil, nil
	}

	return nil, nil
}
