package jfs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
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
		if parts[0] == "remount" {
			hasRemount = true
		} else if parts[0] == "ro" {
			hasRo = true
		} else if parts[0] == "rw" {
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

		if _, ok := jfsKnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("jfs mount option '%s' on line %d is unknown", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if hasEquals && val != "" {
			if _, ok := jfsValuedOptions[key]; !ok && key != "resize" {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("jfs mount option '%s' on line %d does not take a value", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else if hasEquals && val == "" {
			if _, ok := jfsRequireValueOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for jfs mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			} else if _, ok := jfsValuedOptions[key]; !ok && key != "resize" {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("jfs mount option '%s' on line %d does not take a value", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else {
			if _, ok := jfsRequireValueOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for jfs mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}

		switch key {
		case "resize":
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
		case "errors":
			if val != "continue" && val != "remount-ro" && val != "panic" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("jfs mount option 'errors' on line %d has invalid value '%s'. Valid values are 'continue', 'remount-ro', 'panic'", entry.LineNum, val),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "uid", "gid", "umask":
			// Can validate further if needed, but not required yet
		}
	}

	return issues, nil
}

var jfsRequireValueOptions = map[string]struct{}{
	"iocharset": {}, "errors": {}, "context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	"uid": {}, "gid": {}, "umask": {},
}

var jfsValuedOptions = map[string]struct{}{
	"X-mount.file": {},
	"X-mount.group": {},
	"X-mount.mkdir": {},
	"X-mount.mode": {},
	"X-mount.owner": {},
	"comment": {},
	"context": {},
	"defcontext": {},
	"fscontext": {},
	"iocharset": {},
	"rootcontext": {},
	"x-systemd.device-timeout": {},
	"x-systemd.idle-timeout": {},
	"x-systemd.mount-timeout": {},
	"x-systemd.requires": {},
	"x-systemd.requires-mounts-for": {},
	"errors": {},
	"uid": {}, "gid": {}, "umask": {}, "loop": {},
}

var jfsKnownOptions = map[string]struct{}{
	"X-mount.file": {},
	"X-mount.group": {},
	"X-mount.mkdir": {},
	"X-mount.mode": {},
	"X-mount.owner": {},
	"_netdev": {},
	"async": {},
	"atime": {},
	"auto": {},
	"bind": {},
	"context": {},
	"defaults": {},
	"defcontext": {},
	"dev": {},
	"diratime": {},
	"dirsync": {},
	"exec": {},
	"fscontext": {},
	"group": {},
	"iocharset": {},
	"iversion": {},
	"lazytime": {},
	"loud": {},
	"make-private": {},
	"make-runbindable": {},
	"make-shared": {},
	"make-slave": {},
	"make-unbindable": {},
	"mand": {},
	"noatime": {},
	"noauto": {},
	"nodev": {},
	"nodiratime": {},
	"noexec": {},
	"nofail": {},
	"noiversion": {},
	"nolazytime": {},
	"nomand": {},
	"norelatime": {},
	"nostrictatime": {},
	"nosuid": {},
	"nosymfollow": {},
	"nouser": {},
	"owner": {},
	"rbind": {},
	"relatime": {},
	"remount": {},
	"ro": {},
	"rootcontext": {},
	"rw": {},
	"silent": {},
	"strictatime": {},
	"suid": {},
	"sync": {},
	"user": {},
	"users": {},
	"x-systemd.automount": {},
	"x-systemd.device-timeout": {},
	"x-systemd.idle-timeout": {},
	"x-systemd.mount-timeout": {},
	"x-systemd.requires": {},
	"x-systemd.requires-mounts-for": {},
	"resize": {}, "nointegrity": {}, "integrity": {},
	"errors": {}, "noquota": {}, "quota": {}, "usrquota": {}, "grpquota": {},
	"discard": {}, "nodiscard": {}, "uid": {}, "gid": {}, "umask": {}, "comment": {}, "loop": {},
}
