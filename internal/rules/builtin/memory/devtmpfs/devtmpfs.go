package devtmpfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type DevtmpfsRule struct{}

func init() {
	rules.Register(&DevtmpfsRule{})
}

func (r *DevtmpfsRule) Name() string {
	return "devtmpfs"
}

func (r *DevtmpfsRule) Dependencies() []string {
	return []string{}
}

func (r *DevtmpfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "devtmpfs" {
		return nil, nil
	}

	return nil, nil
}
