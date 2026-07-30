package squashfs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type SquashfsRule struct{}

func init() {
	rules.Register(&SquashfsRule{})
}

func (r *SquashfsRule) Name() string {
	return "squashfs"
}

func (r *SquashfsRule) Dependencies() []string {
	return []string{}
}

func (r *SquashfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "squashfs" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "squashfs", squashfsOptions), nil
}

var squashfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"errors": options.EnumOption{
		Allowed: map[string]bool{"continue": true, "panic": true},
	},
	"threads": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}

		if val == "single" || val == "multi" || val == "percpu" {
			return nil
		}

		num, err := strconv.ParseInt(val, 10, 64)
		if err != nil || num <= 0 {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be 'single', 'multi', 'percpu', or an integer > 0", val, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}

		return nil
	}),
	"loop": options.OptionalStringOption{},
})
