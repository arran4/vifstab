package affs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type AffsRule struct{}

func init() {
	rules.Register(&AffsRule{})
}

func (r *AffsRule) Name() string {
	return "affs"
}

func (r *AffsRule) Dependencies() []string {
	return []string{}
}

func (r *AffsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "affs" {
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

		if _, ok := affsKnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown affs mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if val != "" {
			if _, ok := affsValuedOptions[key]; !ok {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("affs mount option '%s' does not take a value on line %d", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else {
			if _, ok := affsValuedOptions[key]; ok {
				if key != "uid" && key != "gid" && key != "setuid" && key != "setgid" { // these can be value-less according to man page (takes UID/GID of calling process)
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Missing value for affs mount option '%s' on line %d", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		}

		switch key {
		case "uid", "gid", "setuid", "setgid":
			if val != "" {
				if num, err := strconv.ParseInt(val, 10, 64); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for affs mount option '%s' on line %d, must be an integer", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				} else {
					if (key == "uid" || key == "setuid") && !api.CheckUserExists(int(num)) {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("User with uid %d does not exist", num),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					}
					if (key == "gid" || key == "setgid") && !api.CheckGroupExists(int(num)) {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Group with gid %d does not exist", num),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					}
				}
			}
		case "mode":
			if val != "" {
				if _, err := strconv.ParseUint(val, 8, 32); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for affs mount option 'mode' on line %d, must be an octal integer", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "reserved", "root":
			if val != "" {
				if _, err := strconv.ParseInt(val, 10, 64); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for affs mount option '%s' on line %d, must be an integer", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "bs":
			if val != "" {
				if val != "512" && val != "1024" && val != "2048" && val != "4096" {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for affs mount option 'bs' on line %d, must be 512, 1024, 2048, or 4096", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		case "volume":
			if val != "" && len(val) > 30 {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for affs mount option 'volume' on line %d, max length is 30", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}
	}

	return issues, nil
}

var affsKnownOptions = map[string]struct{}{
	"uid": {}, "gid": {}, "setuid": {}, "setgid": {}, "mode": {}, "protect": {},
	"usemp": {}, "prefix": {}, "volume": {}, "reserved": {},
	"root": {}, "bs": {}, "grpquota": {}, "noquota": {}, "quota": {}, "usrquota": {},
	// Generic options
	"defaults": {}, "rw": {}, "ro": {}, "noatime": {}, "relatime": {},
	"suid": {}, "nosuid": {}, "exec": {}, "noexec": {}, "auto": {}, "noauto": {},
	"user": {}, "nouser": {}, "sync": {}, "async": {}, "dirsync": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	"nodev": {}, "lazytime": {}, "iversion": {}, "nofail": {},
	"strictatime": {}, "nodiratime": {}, "users": {},
	"_netdev": {}, "dev": {}, "atime": {}, "diratime": {}, "noiversion": {},
	"nolazytime": {}, "owner": {}, "group": {},
	"X-": {}, "beneath": {}, "bind": {}, "exclusive": {}, "fake": {}, "fork": {}, "fstab": {}, "help": {},
	"internal-only": {}, "loud": {}, "make-private": {}, "make-rprivate": {}, "make-rshared": {}, "make-rslave": {}, "make-runbindable": {}, "make-shared": {},
	"make-slave": {}, "make-unbindable": {}, "map-groups": {}, "map-users": {}, "mkdir": {}, "namespace": {}, "no-canonicalize": {}, "no-mtab": {},
	"nomand": {}, "nosymfollow": {}, "onlyonce": {}, "options": {}, "options-mode": {}, "options-source": {}, "options-source-force": {}, "rbind": {},
	"read-only": {}, "read-write": {}, "remount": {}, "show-labels": {}, "silent": {}, "source": {}, "target": {}, "target-prefix": {},
	"test-opts": {}, "types": {}, "uuid": {}, "verbose": {}, "version": {}, "x-": {},
}

var affsValuedOptions = map[string]struct{}{
	"uid": {}, "gid": {}, "setuid": {}, "setgid": {}, "mode": {}, "prefix": {},
	"volume": {}, "reserved": {}, "root": {}, "bs": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	"fstab": {}, "map-groups": {}, "map-users": {}, "mkdir": {}, "namespace": {}, "options": {}, "options-mode": {}, "options-source": {},
	"source": {}, "target": {}, "target-prefix": {}, "test-opts": {}, "types": {}, "uuid": {},
}
