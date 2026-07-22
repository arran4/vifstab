package smackfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type SmackfsRule struct{}

func init() {
	rules.Register(&SmackfsRule{})
}

func (r *SmackfsRule) Name() string {
	return "smackfs"
}

func (r *SmackfsRule) Dependencies() []string {
	return []string{}
}

func (r *SmackfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "smackfs" {
		return nil, nil
	}

	return nil, nil
}
