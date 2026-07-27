package mqueue

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type MqueueRule struct{}

func init() {
	rules.Register(&MqueueRule{})
}

func (r *MqueueRule) Name() string {
	return "mqueue"
}

func (r *MqueueRule) Dependencies() []string {
	return []string{}
}

func (r *MqueueRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "mqueue" {
		return nil, nil
	}

	return nil, nil
}
