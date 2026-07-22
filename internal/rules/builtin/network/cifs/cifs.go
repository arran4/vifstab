package cifs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type CifsRule struct{}

func init() {
	rules.Register(&CifsRule{})
}

func (r *CifsRule) Name() string {
	return "cifs"
}

func (r *CifsRule) Dependencies() []string {
	return []string{}
}

func (r *CifsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "cifs" {
		return nil, nil
	}

	return nil, nil
}
