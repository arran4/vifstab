package ntfs3g

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Ntfs3GRule struct{}

func init() {
	rules.Register(&Ntfs3GRule{})
}

func (r *Ntfs3GRule) Name() string {
	return "ntfs3g"
}

func (r *Ntfs3GRule) Dependencies() []string {
	return []string{}
}

func (r *Ntfs3GRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ntfs3g" {
		return nil, nil
	}

	return nil, nil
}
