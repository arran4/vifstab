package tracefs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type TracefsRule struct{}

func init() {
	rules.Register(&TracefsRule{})
}

func (r *TracefsRule) Name() string {
	return "tracefs"
}

func (r *TracefsRule) Dependencies() []string {
	return []string{}
}

func (r *TracefsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "tracefs" {
		return nil, nil
	}

	return nil, nil
}
