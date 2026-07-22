package configfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type ConfigfsRule struct{}

func init() {
	rules.Register(&ConfigfsRule{})
}

func (r *ConfigfsRule) Name() string {
	return "configfs"
}

func (r *ConfigfsRule) Dependencies() []string {
	return []string{}
}

func (r *ConfigfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "configfs" {
		return nil, nil
	}

	return nil, nil
}
