package orangefs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type OrangefsRule struct{}

func init() {
	rules.Register(&OrangefsRule{})
}

func (r *OrangefsRule) Name() string {
	return "orangefs"
}

func (r *OrangefsRule) Dependencies() []string {
	return []string{}
}

func orangefsOptions() map[string]options.OptionValidator {
	return map[string]options.OptionValidator{
		"acl":          options.NoValueOption{},
		"noacl":        options.NoValueOption{},
		"intr":         options.NoValueOption{},
		"nointr":       options.NoValueOption{},
		"local_lock":   options.NoValueOption{},
		"nolocal_lock": options.NoValueOption{},
	}
}

func (r *OrangefsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "orangefs" {
		return nil, nil
	}

	allOpts := options.MergeOptions(options.CommonOptions(), orangefsOptions())
	issues := options.ValidateOptionsMap(ctx, api, entry, fsType, allOpts)

	return issues, nil
}
