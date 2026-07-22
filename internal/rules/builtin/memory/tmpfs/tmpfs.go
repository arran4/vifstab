package tmpfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type TmpfsRule struct{}

func init() {
	rules.Register(&TmpfsRule{})
}

func (r *TmpfsRule) Name() string {
	return "tmpfs"
}

func (r *TmpfsRule) Dependencies() []string {
	return []string{}
}

func (r *TmpfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "tmpfs" {
		return nil, nil
	}

	return nil, nil
}
