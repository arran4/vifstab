package nfsd

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type NfsdRule struct{}

func init() {
	rules.Register(&NfsdRule{})
}

func (r *NfsdRule) Name() string {
	return "nfsd"
}

func (r *NfsdRule) Dependencies() []string {
	return []string{}
}

func (r *NfsdRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "nfsd" {
		return nil, nil
	}

	return nil, nil
}
