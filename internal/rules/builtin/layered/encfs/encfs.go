package encfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type EncfsRule struct{}

func init() {
	rules.Register(&EncfsRule{})
}

func (r *EncfsRule) Name() string {
	return "encfs"
}

func (r *EncfsRule) Dependencies() []string {
	return []string{}
}

func (r *EncfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "encfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "encfs", encfsOptionsMap), nil
}

var encfsOptionsMap = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"verbose":             options.NoValueOption{},
	"v":                   options.NoValueOption{},
	"config":              options.StringOption{},
	"c":                   options.StringOption{},
	"syslogtag":           options.StringOption{},
	"t":                   options.StringOption{},
	"s":                   options.NoValueOption{},
	"f":                   options.NoValueOption{},
	"annotate":            options.NoValueOption{},
	"standard":            options.NoValueOption{},
	"paranoia":            options.NoValueOption{},
	"insecure":            options.NoValueOption{},
	"reverse":             options.NoValueOption{},
	"reversewrite":        options.NoValueOption{},
	"extpass":             options.StringOption{},
	"stdinpass":           options.NoValueOption{},
	"S":                   options.NoValueOption{},
	"anykey":              options.NoValueOption{},
	"forcedecode":         options.NoValueOption{},
	"require-macs":        options.NoValueOption{},
	"idle":                options.IntOption{},
	"i":                   options.IntOption{},
	"ondemand":            options.NoValueOption{},
	"m":                   options.NoValueOption{},
	"delaymount":          options.NoValueOption{},
	"unmount":             options.NoValueOption{},
	"u":                   options.NoValueOption{},
	"public":              options.NoValueOption{},
	"nocache":             options.NoValueOption{},
	"noattrcache":         options.NoValueOption{},
	"nodatacache":         options.NoValueOption{},
	"no-default-flags":    options.NoValueOption{},
	"fuse-debug":          options.NoValueOption{},
	"d":                   options.NoValueOption{},
	"fuse-help":           options.NoValueOption{},
	"H":                   options.NoValueOption{},
})
