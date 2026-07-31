package udf

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type UdfRule struct{}

func init() {
	rules.Register(&UdfRule{})
}

func (r *UdfRule) Name() string {
	return "udf"
}

func (r *UdfRule) Dependencies() []string {
	return []string{}
}

func (r *UdfRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "udf" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "udf", udfOptions), nil
}

var udfOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"uid": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if val == "forget" {
			return nil
		}
		if !hasVal || val == "" {
			return nil
		}
		num, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for udf mount option '%s' on line %d, must be an integer or 'forget'", val, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if !api.CheckUserExists(int(num)) {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("User with uid %d does not exist", num),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"gid": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if val == "forget" {
			return nil
		}
		if !hasVal || val == "" {
			return nil
		}
		num, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for udf mount option '%s' on line %d, must be an integer or 'forget'", val, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if !api.CheckGroupExists(int(num)) {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Group with gid %d does not exist", num),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"umask":     options.OctalOption{},
	"mode":      options.OctalOption{},
	"dmode":     options.OctalOption{},
	"bs":        options.IntOption{},
	"unhide":    options.NoValueOption{},
	"undelete":  options.NoValueOption{},
	"adinicb":   options.NoValueOption{},
	"noadinicb": options.NoValueOption{},
	"shortad":   options.NoValueOption{},
	"longad":    options.NoValueOption{},
	"nostrict":  options.NoValueOption{},
	"iocharset": options.IOCharsetOption{},
	"novrs":     options.NoValueOption{},
	"session":   options.IntOption{},
	"anchor":    options.IntOption{},
	"lastblock": options.IntOption{},
})
