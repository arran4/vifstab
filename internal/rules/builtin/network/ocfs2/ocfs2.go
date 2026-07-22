package ocfs2

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Ocfs2Rule struct{}

func init() {
	rules.Register(&Ocfs2Rule{})
}

func (r *Ocfs2Rule) Name() string {
	return "ocfs2"
}

func (r *Ocfs2Rule) Dependencies() []string {
	return []string{}
}

func (r *Ocfs2Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ocfs2" {
		return nil, nil
	}

	return nil, nil
}
