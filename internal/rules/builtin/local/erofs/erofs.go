package erofs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type ErofsRule struct{}

func init() {
	rules.Register(&ErofsRule{})
}

func (r *ErofsRule) Name() string {
	return "erofs"
}

func (r *ErofsRule) Dependencies() []string {
	return []string{}
}

func (r *ErofsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "erofs" {
		return nil, nil
	}

	return nil, nil
}
