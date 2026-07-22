package gcsfuse

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type GcsfuseRule struct{}

func init() {
	rules.Register(&GcsfuseRule{})
}

func (r *GcsfuseRule) Name() string {
	return "gcsfuse"
}

func (r *GcsfuseRule) Dependencies() []string {
	return []string{}
}

func (r *GcsfuseRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "gcsfuse" {
		return nil, nil
	}

	return nil, nil
}
