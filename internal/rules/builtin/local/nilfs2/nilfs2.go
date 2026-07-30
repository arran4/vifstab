package nilfs2

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type Nilfs2Rule struct{}

func init() {
	rules.Register(&Nilfs2Rule{})
}

func (r *Nilfs2Rule) Name() string {
	return "nilfs2"
}

func (r *Nilfs2Rule) Dependencies() []string {
	return []string{}
}

func (r *Nilfs2Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "nilfs2" {
		return nil, nil
	}

	options := rules.SplitOptions(entry.Fields[3])
	hasRo := false
	hasRw := false

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		if parts[0] == "ro" {
			hasRo = true
		} else if parts[0] == "rw" {
			hasRw = true
		}
	}

	var issues []rules.ValidationIssue
	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		hasEquals := len(parts) > 1
		if hasEquals {
			val = parts[1]
		}

		validator, ok := nilfs2Options[key]
		if !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("nilfs2 mount option '%s' on line %d is unknown", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if validator != nil {
			issues = append(issues, validator.ValidateOption(ctx, api, entry, "nilfs2", key, val, hasEquals)...)
		}

		if key == "cp" {
			if !hasRo || hasRw {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("nilfs2 mount option 'cp' on line %d requires a read-only mount option ('ro')", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		} else if key == "norecovery" {
			if !hasRo {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("nilfs2 mount option 'norecovery' on line %d will fail for r/w mounts on an unclean volume. Using read-only ('ro') is recommended.", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
		}
	}

	return issues, nil
}

type nilfs2NoValueOption struct{}

func (nilfs2NoValueOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
	if hasVal && val != "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("nilfs2 mount option '%s' on line %d does not take a value", key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}

var nilfs2Options = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"barrier":   nilfs2NoValueOption{},
	"nobarrier": nilfs2NoValueOption{},
	"errors": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for nilfs2 mount option 'errors' on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if val != "continue" && val != "remount-ro" && val != "panic" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("nilfs2 mount option 'errors' on line %d has invalid value '%s'. Valid values are 'continue', 'remount-ro', 'panic'", entry.LineNum, val),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"cp": options.StringOption{},
	"order": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for nilfs2 mount option 'order' on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if val != "relaxed" && val != "strict" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("nilfs2 mount option 'order' on line %d has invalid value '%s'. Valid values are 'relaxed', 'strict'", entry.LineNum, val),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"norecovery": nilfs2NoValueOption{},
	"discard":    nilfs2NoValueOption{},
	"nodiscard":  nilfs2NoValueOption{},
})
