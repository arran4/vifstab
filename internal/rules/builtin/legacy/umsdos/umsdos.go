package umsdos

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/local/fat"
	"github.com/arran4/vifstab/internal/rules/options"
)

type UmsdosRule struct{}

func init() {
	rules.Register(&UmsdosRule{})
}

func (r *UmsdosRule) Name() string {
	return "umsdos"
}

func (r *UmsdosRule) Dependencies() []string {
	return []string{}
}

func (r *UmsdosRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "umsdos" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "umsdos", umsdosOptions), nil
}

var umsdosOptions = options.MergeOptions(fat.Options, map[string]options.OptionValidator{
	"dotsOK": options.IncorrectOption{Reason: "umsdos explicitly kills dotsOK mount option"},
})
