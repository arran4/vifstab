package qnx6

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Qnx6Rule struct{}

func init() {
	rules.Register(&Qnx6Rule{})
}

func (r *Qnx6Rule) Name() string {
	return "qnx6"
}

func (r *Qnx6Rule) Dependencies() []string {
	return []string{}
}

func (r *Qnx6Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "qnx6" {
		return nil, nil
	}

	return nil, nil
}
