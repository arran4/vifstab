package hugetlbfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type HugetlbfsRule struct{}

func init() {
	rules.Register(&HugetlbfsRule{})
}

func (r *HugetlbfsRule) Name() string {
	return "hugetlbfs"
}

func (r *HugetlbfsRule) Dependencies() []string {
	return []string{}
}

func (r *HugetlbfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "hugetlbfs" {
		return nil, nil
	}

	return nil, nil
}
