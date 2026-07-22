package iso9660

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Iso9660Rule struct{}

func init() {
	rules.Register(&Iso9660Rule{})
}

func (r *Iso9660Rule) Name() string {
	return "iso9660"
}

func (r *Iso9660Rule) Dependencies() []string {
	return []string{}
}

func (r *Iso9660Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "iso9660" {
		return nil, nil
	}

	return nil, nil
}
