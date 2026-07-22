package jffs2

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Jffs2Rule struct{}

func init() {
	rules.Register(&Jffs2Rule{})
}

func (r *Jffs2Rule) Name() string {
	return "jffs2"
}

func (r *Jffs2Rule) Dependencies() []string {
	return []string{}
}

func (r *Jffs2Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "jffs2" {
		return nil, nil
	}

	return nil, nil
}
