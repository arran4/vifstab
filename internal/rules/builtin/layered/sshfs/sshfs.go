package sshfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type SshfsRule struct{}

func init() {
	rules.Register(&SshfsRule{})
}

func (r *SshfsRule) Name() string {
	return "sshfs"
}

func (r *SshfsRule) Dependencies() []string {
	return []string{}
}

func (r *SshfsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "sshfs" {
		return nil, nil
	}

	return nil, nil
}
