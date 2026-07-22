package ecryptfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type EcryptfsRule struct{}

func init() {
	rules.Register(&EcryptfsRule{})
}

func (r *EcryptfsRule) Name() string {
	return "ecryptfs"
}

func (r *EcryptfsRule) Dependencies() []string {
	return []string{}
}

func (r *EcryptfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ecryptfs" {
		return nil, nil
	}

	return nil, nil
}
