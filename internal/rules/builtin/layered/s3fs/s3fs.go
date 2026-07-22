package s3fs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type S3fsRule struct{}

func init() {
	rules.Register(&S3fsRule{})
}

func (r *S3fsRule) Name() string {
	return "s3fs"
}

func (r *S3fsRule) Dependencies() []string {
	return []string{}
}

func (r *S3fsRule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "s3fs" {
		return nil, nil
	}

	return nil, nil
}
