package ext2

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Ext2Rule struct{}

func init() {
	rules.Register(&Ext2Rule{})
}

func (r *Ext2Rule) Name() string {
	return "ext2"
}

func (r *Ext2Rule) Dependencies() []string {
	return []string{}
}

func (r *Ext2Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ext2" {
		return nil, nil
	}

	return nil, nil
}
