package tmpfs

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type TmpfsRule struct{}

func init() {
	rules.Register(&TmpfsRule{})
}

func (r *TmpfsRule) Name() string {
	return "tmpfs"
}

func (r *TmpfsRule) Dependencies() []string {
	return []string{}
}

func (r *TmpfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "tmpfs" {
		return nil, nil
	}

	return rules.ValidateOptionsMap(ctx, api, entry, "tmpfs", tmpfsOptions), nil
}

var tmpfsOptions = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"size":      rules.StringOption{},
	"nr_blocks": rules.StringOption{},
	"nr_inodes": rules.StringOption{},
	"noswap":    rules.NoValueOption{},
	"mode":      rules.OctalOption{},
	"gid":       rules.GIDOption{},
	"uid":       rules.UIDOption{},
	"huge": rules.EnumOption{
		Allowed: map[string]bool{"never": true, "always": true, "within_size": true, "advise": true, "deny": true, "force": true},
	},
	"mpol": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if !strings.HasPrefix(val, "default") && !strings.HasPrefix(val, "prefer:") && !strings.HasPrefix(val, "bind:") && !strings.HasPrefix(val, "interleave") && !strings.HasPrefix(val, "local") {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for tmpfs mount option 'mpol' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
})
