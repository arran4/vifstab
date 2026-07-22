package msdos

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type MsdosRule struct{}

func init() {
	rules.Register(&MsdosRule{})
}

func (r *MsdosRule) Name() string {
	return "msdos"
}

func (r *MsdosRule) Dependencies() []string {
	return []string{}
}

func (r *MsdosRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "msdos" {
		return nil, nil
	}

	return nil, nil
}
