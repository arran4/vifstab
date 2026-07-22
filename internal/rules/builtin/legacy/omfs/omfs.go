package omfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type OmfsRule struct{}

func init() {
	rules.Register(&OmfsRule{})
}

func (r *OmfsRule) Name() string {
	return "omfs"
}

func (r *OmfsRule) Dependencies() []string {
	return []string{}
}

func (r *OmfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "omfs" {
		return nil, nil
	}

	return nil, nil
}
