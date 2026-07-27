package udf

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type UdfRule struct{}

func init() {
	rules.Register(&UdfRule{})
}

func (r *UdfRule) Name() string {
	return "udf"
}

func (r *UdfRule) Dependencies() []string {
	return []string{}
}

func (r *UdfRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "udf" {
		return nil, nil
	}

	return nil, nil
}
