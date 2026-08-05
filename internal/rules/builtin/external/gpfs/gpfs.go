package gpfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/external"
	"github.com/arran4/vifstab/internal/rules/options"
)

var gpfsOptionsMap = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"atime":      options.OptionalStringOption{},
	"mtime":      options.OptionalStringOption{},
	"noatime":    options.OptionalStringOption{},
	"nomtime":    options.OptionalStringOption{},
	"norelatime": options.OptionalStringOption{},
	"nosyncnfs":  options.OptionalStringOption{},
	"relatime":   options.OptionalStringOption{},
	"syncnfs":    options.OptionalStringOption{},
	"useNSDserver": options.EnumOption{Allowed: map[string]bool{
		"always":   true,
		"asfound":  true,
		"asneeded": true,
		"never":    true,
	}},
})

type GPFSRule struct{}

func init() {
	rules.Register(&GPFSRule{})
}

func (r *GPFSRule) Name() string {
	return "gpfs"
}

func (r *GPFSRule) Dependencies() []string {
	return []string{}
}

func (r *GPFSRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "gpfs" {
		return nil, nil
	}

	issues := external.CheckModuleSupport(ctx, fsType, entry, "mmfs26", "")

	optIssues := options.ValidateOptionsMap(ctx, api, entry, "gpfs", gpfsOptionsMap)
	issues = append(issues, optIssues...)

	return issues, nil
}
