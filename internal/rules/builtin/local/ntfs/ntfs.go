package ntfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type NtfsRule struct{}

func init() {
	rules.Register(&NtfsRule{})
}

func (r *NtfsRule) Name() string {
	return "ntfs"
}

func (r *NtfsRule) Dependencies() []string {
	return []string{}
}

func (r *NtfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ntfs" {
		return nil, nil
	}

	return nil, nil
}
