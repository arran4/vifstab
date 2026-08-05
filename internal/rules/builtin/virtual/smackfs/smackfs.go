package smackfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type SmackfsRule struct{}

func init() {
	rules.Register(&SmackfsRule{})
}

func (r *SmackfsRule) Name() string {
	return "smackfs"
}

func (r *SmackfsRule) Dependencies() []string {
	return []string{}
}

func (r *SmackfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "smackfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "smackfs", smackfsOptions), nil
}

var smackfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"smackfsdef":       options.StringOption{},
	"smackfsfloor":     options.StringOption{},
	"smackfshat":       options.StringOption{},
	"smackfsroot":      options.StringOption{},
	"smackfstransmute": options.StringOption{},
})
