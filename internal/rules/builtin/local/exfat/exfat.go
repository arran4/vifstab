package exfat

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type ExfatRule struct{}

func init() {
	rules.Register(&ExfatRule{})
}

func (r *ExfatRule) Name() string {
	return "exfat"
}

func (r *ExfatRule) Dependencies() []string {
	return []string{}
}

func (r *ExfatRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "exfat" {
		return nil, nil
	}

	return nil, nil
}
