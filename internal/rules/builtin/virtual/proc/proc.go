package proc

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/kernel"
	"github.com/arran4/vifstab/internal/rules/options"
)

type ProcRule struct{}

func init() {
	rules.Register(&ProcRule{})
}

func (r *ProcRule) Name() string {
	return "proc"
}

func (r *ProcRule) Dependencies() []string {
	return []string{}
}

func (r *ProcRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "proc" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "proc", procOptions), nil
}

var procOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"hidepid": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		var issues []rules.ValidationIssue
		if !hasVal || val == "" {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
		hostVer := kernel.GetHostKernelVersion(api)
		if hostVer.LessThan(3, 3, 0) {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("proc mount option 'hidepid' is only supported on Linux 3.3+ (detected %s)", hostVer),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
		return issues
	}),
	"gid": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		var issues []rules.ValidationIssue
		if !hasVal || val == "" {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
		hostVer := kernel.GetHostKernelVersion(api)
		if hostVer.LessThan(3, 3, 0) {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("proc mount option 'gid' is only supported on Linux 3.3+ (detected %s)", hostVer),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
		return issues
	}),
	"subset": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		var issues []rules.ValidationIssue
		if !hasVal || val == "" {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
		hostVer := kernel.GetHostKernelVersion(api)
		if hostVer.LessThan(5, 8, 0) {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("proc mount option 'subset' is only supported on Linux 5.8+ (detected %s)", hostVer),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
		if val != "pid" && val != "" {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("proc mount option 'subset' expects 'pid', got '%s'", val),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
		return issues
	}),
})
