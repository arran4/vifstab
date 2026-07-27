package overlay

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type OverlayRule struct{}

func init() {
	rules.Register(&OverlayRule{})
}

func (r *OverlayRule) Name() string {
	return "overlay"
}

func (r *OverlayRule) Dependencies() []string {
	return []string{}
}

func (r *OverlayRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "overlay" {
		return nil, nil
	}

	return nil, nil
}
