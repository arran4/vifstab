package v9fs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type V9fsRule struct{}

func init() {
	rules.Register(&V9fsRule{})
}

func (r *V9fsRule) Name() string {
	return "v9fs"
}

func (r *V9fsRule) Dependencies() []string {
	return []string{}
}

func (r *V9fsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "v9fs" {
		return nil, nil
	}

	return nil, nil
}
