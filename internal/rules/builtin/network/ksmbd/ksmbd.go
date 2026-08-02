package ksmbd

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type KsmbdRule struct{}

func init() {
	rules.Register(&KsmbdRule{})
}

func (r *KsmbdRule) Name() string {
	return "ksmbd"
}

func (r *KsmbdRule) Dependencies() []string {
	return []string{}
}

func (r *KsmbdRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ksmbd" {
		return nil, nil
	}

	return []rules.ValidationIssue{
		{
			Message: fmt.Sprintf("ksmbd is a kernel server, not a client filesystem; use 'cifs' or 'smb3' to mount SMB shares on line %d", entry.LineNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		},
	}, nil
}
