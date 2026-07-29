package fat

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type FatRule struct{}

func init() {
	rules.Register(&FatRule{})
}

func (r *FatRule) Name() string {
	return "fat"
}

func (r *FatRule) Dependencies() []string {
	return []string{}
}

func (r *FatRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "fat" {
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

		if _, ok := fatKnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown fat mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if val != "" {
			if _, ok := fatValuedOptions[key]; !ok {
				if _, optional := fatOptionalValuedOptions[key]; !optional {
					if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("fat mount option '%s' does not take a value on line %d", key, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					}
				}
			}
		} else {
			if _, ok := fatValuedOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for fat mount option '%s' on line %d", key, entry.LineNum),
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
						Message: fmt.Sprintf("Invalid value '%s' for fat mount option '%s' on line %d, must be an integer", val, key, entry.LineNum),
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
						Message: fmt.Sprintf("Invalid value '%s' for fat mount option '%s' on line %d, must be an octal integer", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "dos1xfloppy", "dotsOK":
			if val != "" {
				if val != "0" && val != "1" && val != "yes" && val != "no" && val != "true" && val != "false" {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for fat mount option '%s' on line %d, must be a boolean (0, 1, yes, no, true, false)", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "check":
			if val != "" && val != "s" && val != "strict" && val != "r" && val != "relaxed" && val != "n" && val != "normal" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for fat mount option 'check' on line %d (only 's', 'r', or 'n' are supported)", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "tz":
			if val != "" && val != "UTC" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for fat mount option 'tz' on line %d (only 'UTC' is supported)", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "errors":
			if val != "" && val != "panic" && val != "continue" && val != "remount-ro" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for fat mount option 'errors' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "nfs":
			if val != "" && val != "stale_rw" && val != "nostale_ro" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for fat mount option 'nfs' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "blocksize":
			if val != "" && val != "512" && val != "1024" && val != "2048" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for fat mount option 'blocksize' on line %d (only 512, 1024, or 2048 are supported)", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "fat":
			if val != "" && val != "12" && val != "16" && val != "32" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for fat mount option 'fat' on line %d (only 12, 16, or 32 are supported)", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}
	}

	return issues, nil
}

var fatKnownOptions = map[string]struct{}{
	"blocksize": {}, "uid": {}, "gid": {}, "umask": {}, "dmask": {}, "fmask": {},
	"allow_utime": {}, "check": {}, "codepage": {}, "conv": {}, "cvf_format": {},
	"cvf_option": {}, "debug": {}, "discard": {}, "dos1xfloppy": {}, "errors": {},
	"fat": {}, "iocharset": {}, "nfs": {}, "tz": {}, "time_offset": {}, "quiet": {},
	"rodir": {}, "showexec": {}, "sys_immutable": {}, "flush": {}, "usefree": {},
	"dots": {}, "nodots": {}, "dotsOK": {},

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

var fatValuedOptions = map[string]struct{}{
	"blocksize": {}, "uid": {}, "gid": {}, "umask": {}, "dmask": {}, "fmask": {},
	"allow_utime": {}, "check": {}, "codepage": {}, "conv": {}, "cvf_format": {},
	"cvf_option": {}, "errors": {}, "fat": {}, "iocharset": {}, "tz": {}, "time_offset": {},

	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	// Generic valued options added automatically
	"fstab": {}, "map-groups": {}, "map-users": {}, "mkdir": {}, "namespace": {}, "options": {}, "options-mode": {}, "options-source": {},
	"source": {}, "target": {}, "target-prefix": {}, "test-opts": {}, "types": {}, "uuid": {},
}

var fatOptionalValuedOptions = map[string]struct{}{
	"dos1xfloppy": {}, "nfs": {}, "dotsOK": {},
}
