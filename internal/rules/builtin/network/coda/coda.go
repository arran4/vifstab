package coda

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type CodaRule struct{}

func init() {
	rules.Register(&CodaRule{})
}

func (r *CodaRule) Name() string {
	return "coda"
}

func (r *CodaRule) Dependencies() []string {
	return []string{}
}

func (r *CodaRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "coda" {
		return nil, nil
	}

	return nil, nil
}
