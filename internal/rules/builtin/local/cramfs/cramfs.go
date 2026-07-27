package cramfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type CramfsRule struct{}

func init() {
	rules.Register(&CramfsRule{})
}

func (r *CramfsRule) Name() string {
	return "cramfs"
}

func (r *CramfsRule) Dependencies() []string {
	return []string{}
}

func (r *CramfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "cramfs" {
		return nil, nil
	}

	return nil, nil
}
