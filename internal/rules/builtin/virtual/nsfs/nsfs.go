package nsfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type NsfsRule struct{}

func init() {
	rules.Register(&NsfsRule{})
}

func (r *NsfsRule) Name() string {
	return "nsfs"
}

func (r *NsfsRule) Dependencies() []string {
	return []string{}
}

func (r *NsfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "nsfs" {
		return nil, nil
	}

	return nil, nil
}
