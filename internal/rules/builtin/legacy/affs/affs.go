package affs

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
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

	return options.ValidateOptionsMap(ctx, api, entry, "affs", affsOptions), nil
}

var affsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"uid":      options.OptionalUIDOption{},
	"setuid":   options.OptionalUIDOption{},
	"gid":      options.OptionalGIDOption{},
	"setgid":   options.OptionalGIDOption{},
	"mode":     options.OctalOption{},
	"protect":  options.NoValueOption{},
	"usemp":    options.NoValueOption{},
	"prefix":   options.StringOption{},
	"volume": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
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
	"reserved": options.IntOption{},
	"root":     options.IntOption{},
	"bs": options.EnumOption{
		Allowed:      map[string]bool{"512": true, "1024": true, "2048": true, "4096": true},
		CustomErrMsg: "Invalid value '%s' for affs mount option 'bs' on line %d, must be 512, 1024, 2048, or 4096",
	},
	"grpquota": options.NoValueOption{},
	"noquota":  options.NoValueOption{},
	"quota":    options.NoValueOption{},
	"usrquota": options.NoValueOption{},
})
