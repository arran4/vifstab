package jfs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
	"github.com/arran4/vifstab/internal/rules/options"
)

type JfsRule struct{}

func init() {
	rules.Register(&JfsRule{})
}

func (r *JfsRule) Name() string {
	return "jfs"
}

func (r *JfsRule) Dependencies() []string {
	return []string{"linux_fs"}
}

func (r *JfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "jfs" {
		return nil, nil
	}

	issues := legacy.CheckFSVersion(ctx, api, entry, "jfs", kernel.Ver{}, kernel.Ver{}, false, "", true)

	options := rules.SplitOptions(entry.Fields[3])
	hasRemount := false
	hasRo := false
	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		switch parts[0] {
		case "remount":
			hasRemount = true
		case "ro":
			hasRo = true
		case "rw":
			hasRo = false
		}
	}

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		hasEquals := len(parts) > 1
		if hasEquals {
			val = parts[1]
		}

		validator, ok := jfsOptions[key]
		if !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("jfs mount option '%s' on line %d is unknown", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if key == "resize" {
			if !hasRemount || hasRo {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("jfs mount option 'resize' on line %d requires a read-write remount", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			if val != "" {
				if _, err := strconv.Atoi(val); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for resize on line %d, must be an integer", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
			continue
		}

		if validator != nil {
			issues = append(issues, validator.ValidateOption(ctx, api, entry, "jfs", key, val, hasEquals, nil)...)
		}
	}

	return issues, nil
}

type jfsNoValueOption struct{}

func (jfsNoValueOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
	if hasVal {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("jfs mount option '%s' on line %d does not take a value", key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}

var jfsOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"iocharset": options.IOCharsetOption{},
	"errors": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for jfs mount option 'errors' on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if val != "continue" && val != "remount-ro" && val != "panic" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("jfs mount option 'errors' on line %d has invalid value '%s'. Valid values are 'continue', 'remount-ro', 'panic'", entry.LineNum, val),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"resize":        options.OptionalStringOption{},
	"nointegrity":   jfsNoValueOption{},
	"integrity":     jfsNoValueOption{},
	"noquota":       jfsNoValueOption{},
	"quota":         jfsNoValueOption{},
	"usrquota":      jfsNoValueOption{},
	"grpquota":      jfsNoValueOption{},
	"discard":       jfsNoValueOption{},
	"nodiscard":     jfsNoValueOption{},
	"uid":           options.IntOption{},
	"gid":           options.IntOption{},
	"umask":         options.OctalOption{},
	"comment":       options.StringOption{},
	"loop":          options.OptionalStringOption{},
	"mand":          jfsNoValueOption{},
	"norelatime":    jfsNoValueOption{},
	"nostrictatime": jfsNoValueOption{},
})
