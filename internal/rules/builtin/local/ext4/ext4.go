package ext4

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Ext4Rule struct{}

func init() {
	rules.Register(&Ext4Rule{})
}

func (r *Ext4Rule) Name() string {
	return "ext4"
}

func (r *Ext4Rule) Dependencies() []string {
	return []string{}
}

func (r *Ext4Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ext4" {
		return nil, nil
	}

	return nil, nil
}
