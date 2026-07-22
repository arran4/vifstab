package orangefs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type OrangefsRule struct{}

func init() {
	rules.Register(&OrangefsRule{})
}

func (r *OrangefsRule) Name() string {
	return "orangefs"
}

func (r *OrangefsRule) Dependencies() []string {
	return []string{}
}

func (r *OrangefsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "orangefs" {
		return nil, nil
	}

	return nil, nil
}
