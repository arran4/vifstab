package efivarfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type EfivarfsRule struct{}

func init() {
	rules.Register(&EfivarfsRule{})
}

func (r *EfivarfsRule) Name() string {
	return "efivarfs"
}

func (r *EfivarfsRule) Dependencies() []string {
	return []string{}
}

func (r *EfivarfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "efivarfs" {
		return nil, nil
	}

	return nil, nil
}
