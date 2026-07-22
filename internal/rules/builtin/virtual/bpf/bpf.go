package bpf

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type BpfRule struct{}

func init() {
	rules.Register(&BpfRule{})
}

func (r *BpfRule) Name() string {
	return "bpf"
}

func (r *BpfRule) Dependencies() []string {
	return []string{}
}

func (r *BpfRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "bpf" {
		return nil, nil
	}

	return nil, nil
}
