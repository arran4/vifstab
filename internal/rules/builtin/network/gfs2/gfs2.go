package gfs2

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Gfs2Rule struct{}

func init() {
	rules.Register(&Gfs2Rule{})
}

func (r *Gfs2Rule) Name() string {
	return "gfs2"
}

func (r *Gfs2Rule) Dependencies() []string {
	return []string{}
}

func (r *Gfs2Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "gfs2" {
		return nil, nil
	}

	return nil, nil
}
