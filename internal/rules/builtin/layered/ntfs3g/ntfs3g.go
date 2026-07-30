package ntfs3g

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type Ntfs3GRule struct{}

func init() {
	rules.Register(&Ntfs3GRule{})
}

func (r *Ntfs3GRule) Name() string {
	return "ntfs-3g"
}

func (r *Ntfs3GRule) Dependencies() []string {
	return []string{}
}

func (r *Ntfs3GRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ntfs-3g" && fsType != "lowntfs-3g" {
		return nil, nil
	}

	return options.ValidateOptionsMap(ctx, api, entry, "ntfs-3g", ntfs3gOptions), nil
}

var ntfs3gOptions = options.MergeOptions(options.CommonOptions(), map[string]options.OptionValidator{
	"acl":         options.NoValueOption{},
	"allow_other": options.NoValueOption{},
	"big_writes":  options.NoValueOption{},
	"compression": options.NoValueOption{},
	"debug":       options.NoValueOption{},
	"delay_mtime": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if hasVal && val != "" {
			return (options.IntOption{}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal, nil)
		}
		return nil
	}),
	"dmask":          options.OctalOption{CustomErrFormat: "must be an octal value"},
	"efs_raw":        options.NoValueOption{},
	"fmask":          options.OctalOption{CustomErrFormat: "must be an octal value"},
	"force":          options.NoValueOption{},
	"hide_dot_files": options.NoValueOption{},
	"hide_hid_files": options.NoValueOption{},
	"ignore_case": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if entry.Fields[2] != "lowntfs-3g" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("ignore_case is only supported with lowntfs-3g on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"inherit":          options.NoValueOption{},
	"locale":           options.StringOption{},
	"max_read":         options.IntOption{},
	"no_def_opts":      options.NoValueOption{},
	"no_detach":        options.NoValueOption{},
	"nocompression":    options.NoValueOption{},
	"norecover":        options.NoValueOption{},
	"permissions":      options.NoValueOption{},
	"posix_nlink":      options.NoValueOption{},
	"recover":          options.NoValueOption{},
	"remove_hiberfile": options.NoValueOption{},
	"show_sys_files":   options.NoValueOption{},
	"special_files": options.EnumOption{
		Allowed:      map[string]bool{"interix": true, "wsl": true},
		CustomErrMsg: "Invalid value '%s' for ntfs-3g mount option 'special_files' on line %d (must be 'interix' or 'wsl')",
	},
	"streams_interface": options.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
		if !hasVal || val == "" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if val != "none" && val != "windows" && val != "xattr" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for ntfs-3g mount option 'streams_interface' on line %d (must be 'none', 'windows', or 'xattr')", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		if val == "windows" && entry.Fields[2] == "lowntfs-3g" {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("streams_interface=windows is not supported with lowntfs-3g on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}),
	"uid": options.UIDOption{
		CustomUserNotFound: "User ID '%s' specified in 'uid' option on line %d does not exist",
		Level:              rules.IssueLevelWarning,
	},
	"gid": options.GIDOption{
		CustomGroupNotFound: "Group ID '%s' specified in 'gid' option on line %d does not exist",
		Level:               rules.IssueLevelWarning,
	},
	"umask":         options.OctalOption{CustomErrFormat: "must be an octal value"},
	"usermapping":   options.StringOption{},
	"user_xattr":    options.NoValueOption{},
	"windows_names": options.NoValueOption{},
})
