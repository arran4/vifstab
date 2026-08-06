package msdos

import (
	"context"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/local/fat"
	"github.com/arran4/vifstab/internal/rules/options"
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

func (r *MsdosRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) >= 3 && entry.Fields[2] == "msdos" {
		return options.ValidateOptionsMap(ctx, api, entry, "msdos", fat.Options), nil
	}
	return nil, nil
}
