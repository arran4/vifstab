package virtiofs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type VirtiofsRule struct{}

func init() {
	rules.Register(&VirtiofsRule{})
}

func (r *VirtiofsRule) Name() string {
	return "virtiofs"
}

func (r *VirtiofsRule) Dependencies() []string {
	return []string{}
}

func (r *VirtiofsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "virtiofs" {
		return nil, nil
	}

	return nil, nil
}
