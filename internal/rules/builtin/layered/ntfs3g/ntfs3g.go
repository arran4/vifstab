package ntfs3g

import (
	"context"
	"fmt"
	"strconv"
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

	var issues []rules.ValidationIssue
	options := rules.SplitOptions(entry.Fields[3])

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		if len(parts) > 1 {
			val = parts[1]
		}

		if _, ok := ntfs3gKnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown ntfs-3g mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if val != "" {
			if _, ok := ntfs3gValuedOptions[key]; !ok && key != "delay_mtime" {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("ntfs-3g mount option '%s' does not take a value on line %d", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else {
			if _, ok := ntfs3gValuedOptions[key]; ok && key != "delay_mtime" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for ntfs-3g mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}

		switch key {
		case "uid":
			if val != "" {
				if uid, err := strconv.Atoi(val); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for ntfs-3g mount option 'uid' on line %d, must be an integer", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				} else {
					if !api.CheckUserExists(uid) {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("User ID '%s' specified in 'uid' option on line %d does not exist", val, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelWarning,
						})
					}
				}
			}
		case "gid":
			if val != "" {
				if gid, err := strconv.Atoi(val); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for ntfs-3g mount option 'gid' on line %d, must be an integer", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				} else {
					if !api.CheckGroupExists(gid) {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Group ID '%s' specified in 'gid' option on line %d does not exist", val, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelWarning,
						})
					}
				}
			}
		case "dmask", "fmask", "umask":
			if val != "" {
				if _, err := strconv.ParseUint(val, 8, 32); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for ntfs-3g mount option '%s' on line %d, must be an octal value", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "delay_mtime", "max_read":
			if val != "" {
				if _, err := strconv.Atoi(val); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for ntfs-3g mount option '%s' on line %d, must be an integer", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "special_files":
			if val != "interix" && val != "wsl" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ntfs-3g mount option 'special_files' on line %d (must be 'interix' or 'wsl')", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "streams_interface":
			if val != "none" && val != "windows" && val != "xattr" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ntfs-3g mount option 'streams_interface' on line %d (must be 'none', 'windows', or 'xattr')", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			} else if val == "windows" && fsType == "lowntfs-3g" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("streams_interface=windows is not supported with lowntfs-3g on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "ignore_case":
			if fsType != "lowntfs-3g" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("ignore_case is only supported with lowntfs-3g on line %d", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}
	}

	return issues, nil
}

var ntfs3gKnownOptions = map[string]struct{}{
	"acl":               {},
	"allow_other":       {},
	"atime":             {},
	"noatime":           {},
	"relatime":          {},
	"big_writes":        {},
	"compression":       {},
	"debug":             {},
	"delay_mtime":       {},
	"dmask":             {},
	"efs_raw":           {},
	"fmask":             {},
	"force":             {},
	"hide_dot_files":    {},
	"hide_hid_files":    {},
	"ignore_case":       {},
	"inherit":           {},
	"locale":            {},
	"max_read":          {},
	"no_def_opts":       {},
	"no_detach":         {},
	"nocompression":     {},
	"norecover":         {},
	"permissions":       {},
	"posix_nlink":       {},
	"recover":           {},
	"remove_hiberfile":  {},
	"ro":                {},
	"rw":                {},
	"show_sys_files":    {},
	"silent":            {},
	"special_files":     {},
	"streams_interface": {},
	"uid":               {},
	"gid":               {},
	"umask":             {},
	"usermapping":       {},
	"user_xattr":        {},
	"windows_names":     {},

	// Common generic ones added automatically
	"X-": {}, "beneath": {}, "bind": {}, "exclusive": {}, "fake": {}, "fork": {}, "fstab": {}, "help": {},
	"internal-only": {}, "loud": {}, "make-private": {}, "make-rprivate": {}, "make-rshared": {}, "make-rslave": {}, "make-runbindable": {}, "make-shared": {},
	"make-slave": {}, "make-unbindable": {}, "map-groups": {}, "map-users": {}, "mkdir": {}, "namespace": {}, "no-canonicalize": {}, "no-mtab": {},
	"nomand": {}, "nosymfollow": {}, "onlyonce": {}, "options": {}, "options-mode": {}, "options-source": {}, "options-source-force": {}, "rbind": {},
	"read-only": {}, "read-write": {}, "remount": {}, "show-labels": {}, "source": {}, "target": {}, "target-prefix": {},
	"test-opts": {}, "types": {}, "uuid": {}, "verbose": {}, "version": {}, "x-": {},
	"defaults": {}, "suid": {}, "nosuid": {}, "exec": {}, "noexec": {}, "auto": {}, "noauto": {},
	"user": {}, "nouser": {}, "sync": {}, "async": {}, "dirsync": {}, "nodev": {}, "lazytime": {},
	"nofail": {}, "strictatime": {}, "nodiratime": {}, "users": {}, "_netdev": {}, "dev": {}, "diratime": {},
	"noiversion": {}, "nolazytime": {}, "owner": {}, "group": {}, "iversion": {}, "context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
}

var ntfs3gValuedOptions = map[string]struct{}{
	"delay_mtime":       {},
	"dmask":             {},
	"fmask":             {},
	"locale":            {},
	"max_read":          {},
	"special_files":     {},
	"streams_interface": {},
	"uid":               {},
	"gid":               {},
	"umask":             {},
	"usermapping":       {},
	// Generic valued options
	"fstab": {}, "map-groups": {}, "map-users": {}, "mkdir": {}, "namespace": {}, "options": {}, "options-mode": {}, "options-source": {},
	"source": {}, "target": {}, "target-prefix": {}, "test-opts": {}, "types": {}, "uuid": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
}
