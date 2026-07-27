package hpfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type HpfsRule struct{}

func init() {
	rules.Register(&HpfsRule{})
}

func (r *HpfsRule) Name() string {
	return "hpfs"
}

func (r *HpfsRule) Dependencies() []string {
	return []string{}
}

func (r *HpfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "hpfs" {
		return nil, nil
	}

	return nil, nil
}
