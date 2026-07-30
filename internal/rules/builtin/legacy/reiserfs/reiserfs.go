package reiserfs

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
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

	optsIssues := rules.ValidateOptionsMap(ctx, api, entry, "reiserfs", reiserfsOptions)
	return append(issues, optsIssues...), nil
}

var reiserfsOptions = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"conv":                   rules.NoValueOption{},
	"hash":                   rules.EnumOption{Allowed: map[string]bool{"rupasov": true, "tea": true, "r5": true, "detect": true}},
	"hashed_relocation":      rules.NoValueOption{},
	"no_unhashed_relocation": rules.NoValueOption{},
	"noborder":               rules.NoValueOption{},
	"nolog":                  rules.NoValueOption{},
	"notail":                 rules.NoValueOption{},
	"replayonly":             rules.NoValueOption{},
	"resize":                 rules.IntOption{},
	"user_xattr":             rules.NoValueOption{},
	"acl":                    rules.NoValueOption{},
	"barrier":                rules.EnumOption{Allowed: map[string]bool{"none": true, "flush": true}},
})
