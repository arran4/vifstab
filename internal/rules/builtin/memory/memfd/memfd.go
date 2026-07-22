package memfd

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type MemfdRule struct{}

func init() {
	rules.Register(&MemfdRule{})
}

func (r *MemfdRule) Name() string {
	return "memfd"
}

func (r *MemfdRule) Dependencies() []string {
	return []string{}
}

func (r *MemfdRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "memfd" {
		return nil, nil
	}

	return nil, nil
}
