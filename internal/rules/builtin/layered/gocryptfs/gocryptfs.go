package gocryptfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type GocryptfsRule struct{}

func init() {
	rules.Register(&GocryptfsRule{})
}

func (r *GocryptfsRule) Name() string {
	return "gocryptfs"
}

func (r *GocryptfsRule) Dependencies() []string {
	return []string{}
}

func (r *GocryptfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "gocryptfs" {
		return nil, nil
	}

	return nil, nil
}
