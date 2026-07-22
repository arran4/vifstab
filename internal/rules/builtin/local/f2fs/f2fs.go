package f2fs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type F2fsRule struct{}

func init() {
	rules.Register(&F2fsRule{})
}

func (r *F2fsRule) Name() string {
	return "f2fs"
}

func (r *F2fsRule) Dependencies() []string {
	return []string{}
}

func (r *F2fsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "f2fs" {
		return nil, nil
	}

	return nil, nil
}
