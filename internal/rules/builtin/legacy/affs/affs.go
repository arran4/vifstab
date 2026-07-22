package affs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type AffsRule struct{}

func init() {
	rules.Register(&AffsRule{})
}

func (r *AffsRule) Name() string {
	return "affs"
}

func (r *AffsRule) Dependencies() []string {
	return []string{}
}

func (r *AffsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "affs" {
		return nil, nil
	}

	return nil, nil
}
