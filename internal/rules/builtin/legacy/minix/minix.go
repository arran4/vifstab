package minix

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type MinixRule struct{}

func init() {
	rules.Register(&MinixRule{})
}

func (r *MinixRule) Name() string {
	return "minix"
}

func (r *MinixRule) Dependencies() []string {
	return []string{}
}

func (r *MinixRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "minix" {
		return nil, nil
	}

	return nil, nil
}
