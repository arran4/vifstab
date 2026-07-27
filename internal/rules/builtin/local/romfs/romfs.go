package romfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type RomfsRule struct{}

func init() {
	rules.Register(&RomfsRule{})
}

func (r *RomfsRule) Name() string {
	return "romfs"
}

func (r *RomfsRule) Dependencies() []string {
	return []string{}
}

func (r *RomfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "romfs" {
		return nil, nil
	}

	return nil, nil
}
