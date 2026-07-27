package qnx4

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Qnx4Rule struct{}

func init() {
	rules.Register(&Qnx4Rule{})
}

func (r *Qnx4Rule) Name() string {
	return "qnx4"
}

func (r *Qnx4Rule) Dependencies() []string {
	return []string{}
}

func (r *Qnx4Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "qnx4" {
		return nil, nil
	}

	return nil, nil
}
