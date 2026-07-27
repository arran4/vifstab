package hostfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type HostfsRule struct{}

func init() {
	rules.Register(&HostfsRule{})
}

func (r *HostfsRule) Name() string {
	return "hostfs"
}

func (r *HostfsRule) Dependencies() []string {
	return []string{}
}

func (r *HostfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "hostfs" {
		return nil, nil
	}

	return nil, nil
}
