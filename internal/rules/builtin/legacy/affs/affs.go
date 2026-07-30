package affs

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type AffsRule struct{}

func init() {
	rules.Register(&AffsRule{})
}

func (r *AffsRule) Name() string {
	return "affs"
}

func (r *AffsRule) Dependencies() []string {
	return []string{}
}

func (r *AffsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "affs" {
		return nil, nil
	}

	return rules.ValidateOptionsMap(ctx, api, entry, "affs", affsOptions), nil
}

var affsOptions = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"uid":      rules.OptionalUIDOption{},
	"setuid":   rules.OptionalUIDOption{},
	"gid":      rules.OptionalGIDOption{},
	"setgid":   rules.OptionalGIDOption{},
	"mode":     rules.OctalOption{},
	"protect":  rules.NoValueOption{},
	"usemp":    rules.NoValueOption{},
	"prefix":   rules.StringOption{},
	"volume": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if len(val) > 30 {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for affs mount option 'volume' on line %d, max length is 30", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"reserved": rules.IntOption{},
	"root":     rules.IntOption{},
	"bs": rules.EnumOption{
		Allowed:      map[string]bool{"512": true, "1024": true, "2048": true, "4096": true},
		CustomErrMsg: "Invalid value '%s' for affs mount option 'bs' on line %d, must be 512, 1024, 2048, or 4096",
	},
	"grpquota": rules.NoValueOption{},
	"noquota":  rules.NoValueOption{},
	"quota":    rules.NoValueOption{},
	"usrquota": rules.NoValueOption{},
})
