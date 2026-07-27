package glusterfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type GlusterfsRule struct{}

func init() {
	rules.Register(&GlusterfsRule{})
}

func (r *GlusterfsRule) Name() string {
	return "glusterfs"
}

func (r *GlusterfsRule) Dependencies() []string {
	return []string{}
}

func (r *GlusterfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "glusterfs" {
		return nil, nil
	}

	return nil, nil
}
