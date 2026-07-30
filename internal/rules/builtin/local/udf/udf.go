package udf

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
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

	return rules.ValidateOptionsMap(ctx, api, entry, "udf", udfOptions), nil
}

var udfOptions = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"uid": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
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
	"gid": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
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
	"umask":     rules.OctalOption{},
	"mode":      rules.OctalOption{},
	"dmode":     rules.OctalOption{},
	"bs":        rules.IntOption{},
	"unhide":    rules.NoValueOption{},
	"undelete":  rules.NoValueOption{},
	"adinicb":   rules.NoValueOption{},
	"noadinicb": rules.NoValueOption{},
	"shortad":   rules.NoValueOption{},
	"longad":    rules.NoValueOption{},
	"nostrict":  rules.NoValueOption{},
	"iocharset": rules.StringOption{},
	"novrs":     rules.NoValueOption{},
	"session":   rules.IntOption{},
	"anchor":    rules.IntOption{},
	"lastblock": rules.IntOption{},
})
