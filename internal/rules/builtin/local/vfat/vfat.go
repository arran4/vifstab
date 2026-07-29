package vfat

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type VfatRule struct{}

func init() {
	rules.Register(&VfatRule{})
}

func (r *VfatRule) Name() string {
	return "vfat"
}

func (r *VfatRule) Dependencies() []string {
	return []string{}
}

func (r *VfatRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "vfat" {
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

		if _, ok := vfatKnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown vfat mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if val != "" {
			if _, ok := vfatValuedOptions[key]; !ok {
				if _, optional := vfatOptionalValuedOptions[key]; !optional {
					if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("vfat mount option '%s' does not take a value on line %d", key, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					}
				}
			}
		} else {
			if _, ok := vfatValuedOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for vfat mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}

		switch key {
		case "uid", "gid", "time_offset", "codepage":
			if val != "" {
				if num, err := strconv.ParseInt(val, 10, 64); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for vfat mount option '%s' on line %d, must be an integer", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				} else {
					if key == "uid" {
						if !api.CheckUserExists(int(num)) {
							issues = append(issues, rules.ValidationIssue{
								Message: fmt.Sprintf("User with uid %d does not exist", num),
								Line:    entry.RawLine,
								Level:   rules.IssueLevelError,
							})
						}
					} else if key == "gid" {
						if !api.CheckGroupExists(int(num)) {
							issues = append(issues, rules.ValidationIssue{
								Message: fmt.Sprintf("Group with gid %d does not exist", num),
								Line:    entry.RawLine,
								Level:   rules.IssueLevelError,
							})
						}
					}
				}
			}
		case "umask", "dmask", "fmask", "allow_utime":
			if val != "" {
				if _, err := strconv.ParseUint(val, 8, 32); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for vfat mount option '%s' on line %d, must be an octal integer", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "utf8", "uni_xlate", "nonumtail", "dos1xfloppy":
			if val != "" {
				if val != "0" && val != "1" && val != "yes" && val != "no" && val != "true" && val != "false" {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for vfat mount option '%s' on line %d, must be a boolean (0, 1, yes, no, true, false)", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "check":
			if val != "s" && val != "r" && val != "n" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for vfat mount option 'check' on line %d (only 's', 'r', or 'n' are supported)", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "shortname":
			if val != "lower" && val != "win95" && val != "winnt" && val != "mixed" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for vfat mount option 'shortname' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "tz":
			if val != "UTC" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for vfat mount option 'tz' on line %d (only 'UTC' is supported)", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "errors":
			if val != "panic" && val != "continue" && val != "remount-ro" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for vfat mount option 'errors' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "nfs":
			if val != "stale_rw" && val != "nostale_ro" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for vfat mount option 'nfs' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}
	}

	return issues, nil
}

var vfatKnownOptions = map[string]struct{}{
	"uid": {}, "gid": {}, "umask": {}, "dmask": {}, "fmask": {},
	"allow_utime": {}, "codepage": {}, "iocharset": {}, "utf8": {},
	"uni_xlate": {}, "nonumtail": {}, "usefree": {}, "quiet": {},
	"check": {}, "nocase": {}, "shortname": {}, "tz": {},
	"time_offset": {}, "showexec": {}, "debug": {}, "sys_immutable": {},
	"flush": {}, "rodir": {}, "errors": {}, "discard": {},
	"nfs": {}, "dos1xfloppy": {},

	// Generic ones
	"defaults": {}, "rw": {}, "ro": {}, "noatime": {}, "relatime": {},
	"suid": {}, "nosuid": {}, "exec": {}, "noexec": {}, "auto": {}, "noauto": {},
	"user": {}, "nouser": {}, "sync": {}, "async": {}, "dirsync": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	"nodev": {}, "lazytime": {}, "iversion": {}, "nofail": {},
	"strictatime": {}, "nodiratime": {}, "users": {},

	"_netdev": {}, "dev": {}, "atime": {}, "diratime": {}, "noiversion": {},
	"nolazytime": {}, "owner": {}, "group": {},
	// Common generic ones added automatically
	"X-": {}, "beneath": {}, "bind": {}, "exclusive": {}, "fake": {}, "fork": {}, "fstab": {}, "help": {},
	"internal-only": {}, "loud": {}, "make-private": {}, "make-rprivate": {}, "make-rshared": {}, "make-rslave": {}, "make-runbindable": {}, "make-shared": {},
	"make-slave": {}, "make-unbindable": {}, "map-groups": {}, "map-users": {}, "mkdir": {}, "namespace": {}, "no-canonicalize": {}, "no-mtab": {},
	"nomand": {}, "nosymfollow": {}, "onlyonce": {}, "options": {}, "options-mode": {}, "options-source": {}, "options-source-force": {}, "rbind": {},
	"read-only": {}, "read-write": {}, "remount": {}, "show-labels": {}, "silent": {}, "source": {}, "target": {}, "target-prefix": {},
	"test-opts": {}, "types": {}, "uuid": {}, "verbose": {}, "version": {}, "x-": {},
}

var vfatValuedOptions = map[string]struct{}{
	"uid": {}, "gid": {}, "umask": {}, "dmask": {}, "fmask": {},
	"allow_utime": {}, "codepage": {}, "iocharset": {},
	"check": {}, "shortname": {},
	"tz": {}, "time_offset": {}, "errors": {},

	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	// Generic valued options added automatically
	"fstab": {}, "map-groups": {}, "map-users": {}, "mkdir": {}, "namespace": {}, "options": {}, "options-mode": {}, "options-source": {},
	"source": {}, "target": {}, "target-prefix": {}, "test-opts": {}, "types": {}, "uuid": {},
}

var vfatOptionalValuedOptions = map[string]struct{}{
	"utf8": {}, "uni_xlate": {}, "nonumtail": {}, "dos1xfloppy": {}, "nfs": {},
}
