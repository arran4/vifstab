package dlmfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type DlmfsRule struct{}

func init() {
	rules.Register(&DlmfsRule{})
}

func (r *DlmfsRule) Name() string {
	return "dlmfs"
}

func (r *DlmfsRule) Dependencies() []string {
	return []string{}
}

func (r *DlmfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "dlmfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "dlmfs", dlmfsOptions), nil
}

var dlmfsOptions = options.CommonOptions()
