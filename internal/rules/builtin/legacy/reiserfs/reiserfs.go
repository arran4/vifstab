package reiserfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
	"github.com/arran4/vifstab/internal/rules/options"
)

type ReiserfsRule struct{}

func init() {
	rules.Register(&ReiserfsRule{})
}

func (r *ReiserfsRule) Name() string {
	return "reiserfs"
}

func (r *ReiserfsRule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *ReiserfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "reiserfs" {
		return nil, nil
	}

	// ReiserFS was removed in 6.13
	issues := legacy.CheckFSVersion(ctx, api, entry, "reiserfs", kernel.Ver{}, kernel.Ver{Major: 6, Minor: 13, Patch: 0}, false, "", false)

	optsIssues := options.ValidateOptionsMap(ctx, api, entry, "reiserfs", reiserfsOptions)
	return append(issues, optsIssues...), nil
}

var reiserfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"conv":                   options.NoValueOption{},
	"hash":                   options.EnumOption{Allowed: map[string]bool{"rupasov": true, "tea": true, "r5": true, "detect": true}},
	"hashed_relocation":      options.NoValueOption{},
	"no_unhashed_relocation": options.NoValueOption{},
	"noborder":               options.NoValueOption{},
	"nolog":                  options.NoValueOption{},
	"notail":                 options.NoValueOption{},
	"replayonly":             options.NoValueOption{},
	"resize":                 options.IntOption{},
	"user_xattr":             options.NoValueOption{},
	"acl":                    options.NoValueOption{},
	"barrier":                options.EnumOption{Allowed: map[string]bool{"none": true, "flush": true}},
})
