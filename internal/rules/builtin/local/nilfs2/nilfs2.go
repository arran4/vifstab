package nilfs2

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Nilfs2Rule struct{}

func init() {
	rules.Register(&Nilfs2Rule{})
}

func (r *Nilfs2Rule) Name() string {
	return "nilfs2"
}

func (r *Nilfs2Rule) Dependencies() []string {
	return []string{}
}

func (r *Nilfs2Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "nilfs2" {
		return nil, nil
	}

	return nil, nil
}
