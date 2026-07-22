package devpts

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type DevptsRule struct{}

func init() {
	rules.Register(&DevptsRule{})
}

func (r *DevptsRule) Name() string {
	return "devpts"
}

func (r *DevptsRule) Dependencies() []string {
	return []string{}
}

func (r *DevptsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "devpts" {
		return nil, nil
	}

	return nil, nil
}
