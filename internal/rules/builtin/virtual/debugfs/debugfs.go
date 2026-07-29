package debugfs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type DebugfsRule struct{}

func init() {
	rules.Register(&DebugfsRule{})
}

func (r *DebugfsRule) Name() string {
	return "debugfs"
}

func (r *DebugfsRule) Dependencies() []string {
	return []string{}
}

func (r *DebugfsRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "debugfs" {
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

		if _, ok := debugfsKnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown debugfs mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if val != "" {
			if _, ok := debugfsValuedOptions[key]; !ok {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("debugfs mount option '%s' does not take a value on line %d", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else {
			if _, ok := debugfsValuedOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for debugfs mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}

		switch key {
		case "uid", "gid":
			if val != "" {
				if num, err := strconv.ParseInt(val, 10, 64); err != nil {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("Invalid value '%s' for debugfs mount option '%s' on line %d, must be an integer", val, key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				} else {
					if key == "uid" && !api.CheckUserExists(int(num)) {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("User with uid %d does not exist", num),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					}
					if key == "gid" && !api.CheckGroupExists(int(num)) {
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
						Message: fmt.Sprintf("Invalid value '%s' for debugfs mount option 'mode' on line %d, must be an octal integer", val, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		}
	}

	return issues, nil
}

var debugfsKnownOptions = map[string]struct{}{
	"uid": {}, "gid": {}, "mode": {},
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

var debugfsValuedOptions = map[string]struct{}{
	"uid": {}, "gid": {}, "mode": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	"fstab": {}, "map-groups": {}, "map-users": {}, "mkdir": {}, "namespace": {}, "options": {}, "options-mode": {}, "options-source": {},
	"source": {}, "target": {}, "target-prefix": {}, "test-opts": {}, "types": {}, "uuid": {},
}
