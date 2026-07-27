package ntfs3

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Ntfs3Rule struct{}

func init() {
	rules.Register(&Ntfs3Rule{})
}

func (r *Ntfs3Rule) Name() string {
	return "ntfs3"
}

func (r *Ntfs3Rule) Dependencies() []string {
	return []string{}
}

func (r *Ntfs3Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ntfs3" {
		return nil, nil
	}

	return nil, nil
}
