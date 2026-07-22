package squashfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type SquashfsRule struct{}

func init() {
	rules.Register(&SquashfsRule{})
}

func (r *SquashfsRule) Name() string {
	return "squashfs"
}

func (r *SquashfsRule) Dependencies() []string {
	return []string{}
}

func (r *SquashfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "squashfs" {
		return nil, nil
	}

	return nil, nil
}
