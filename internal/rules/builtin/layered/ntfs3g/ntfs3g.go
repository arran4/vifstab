package ntfs3g

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
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

	return rules.ValidateOptionsMap(ctx, api, entry, "ntfs-3g", ntfs3gOptions), nil
}

var ntfs3gOptions = rules.MergeOptions(rules.CommonOptions(), map[string]rules.OptionValidator{
	"acl":         rules.NoValueOption{},
	"allow_other": rules.NoValueOption{},
	"big_writes":  rules.NoValueOption{},
	"compression": rules.NoValueOption{},
	"debug":       rules.NoValueOption{},
	"delay_mtime": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
		if hasVal && val != "" {
			return (rules.IntOption{}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal)
		}
		return nil
	}),
	"dmask":          rules.OctalOption{CustomErrFormat: "must be an octal value"},
	"efs_raw":        rules.NoValueOption{},
	"fmask":          rules.OctalOption{CustomErrFormat: "must be an octal value"},
	"force":          rules.NoValueOption{},
	"hide_dot_files": rules.NoValueOption{},
	"hide_hid_files": rules.NoValueOption{},
	"ignore_case": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
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
	"inherit":          rules.NoValueOption{},
	"locale":           rules.StringOption{},
	"max_read":         rules.IntOption{},
	"no_def_opts":      rules.NoValueOption{},
	"no_detach":        rules.NoValueOption{},
	"nocompression":    rules.NoValueOption{},
	"norecover":        rules.NoValueOption{},
	"permissions":      rules.NoValueOption{},
	"posix_nlink":      rules.NoValueOption{},
	"recover":          rules.NoValueOption{},
	"remove_hiberfile": rules.NoValueOption{},
	"show_sys_files":   rules.NoValueOption{},
	"special_files": rules.EnumOption{
		Allowed:      map[string]bool{"interix": true, "wsl": true},
		CustomErrMsg: "Invalid value '%s' for ntfs-3g mount option 'special_files' on line %d (must be 'interix' or 'wsl')",
	},
	"streams_interface": rules.OptionValidatorFunc(func(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool) []rules.ValidationIssue {
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
	"uid": rules.UIDOption{
		CustomUserNotFound: "User ID '%s' specified in 'uid' option on line %d does not exist",
		Level:              rules.IssueLevelWarning,
	},
	"gid": rules.GIDOption{
		CustomGroupNotFound: "Group ID '%s' specified in 'gid' option on line %d does not exist",
		Level:               rules.IssueLevelWarning,
	},
	"umask":         rules.OctalOption{CustomErrFormat: "must be an octal value"},
	"usermapping":   rules.StringOption{},
	"user_xattr":    rules.NoValueOption{},
	"windows_names": rules.NoValueOption{},
})
