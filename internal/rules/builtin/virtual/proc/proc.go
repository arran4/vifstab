package proc

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type ProcRule struct{}

func init() {
	rules.Register(&ProcRule{})
}

func (r *ProcRule) Name() string {
	return "proc"
}

func (r *ProcRule) Dependencies() []string {
	return []string{}
}

func (r *ProcRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "proc" {
		return nil, nil
	}

	return nil, nil
}
