package ubifs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type UbifsRule struct{}

func init() {
	rules.Register(&UbifsRule{})
}

func (r *UbifsRule) Name() string {
	return "ubifs"
}

func (r *UbifsRule) Dependencies() []string {
	return []string{}
}

func (r *UbifsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ubifs" {
		return nil, nil
	}

	return nil, nil
}
