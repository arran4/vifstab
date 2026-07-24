package ext2

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Ext2Rule struct{}

func init() {
	rules.Register(&Ext2Rule{})
}

func (r *Ext2Rule) Name() string {
	return "ext2"
}

func (r *Ext2Rule) Dependencies() []string {
	return []string{}
}

func splitOptions(optsStr string) []string {
	if optsStr == "" {
		return nil
	}
	opts := strings.Split(optsStr, ",")
	return opts
}

func (r *Ext2Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ext2" {
		return nil, nil
	}

	var issues []rules.ValidationIssue
	options := splitOptions(entry.Fields[3])

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		if len(parts) > 1 {
			val = parts[1]
		}

		if _, ok := ext2KnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown ext2 mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if val != "" {
			if _, ok := ext2ValuedOptions[key]; !ok {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("ext2 mount option '%s' does not take a value on line %d", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else {
			if _, ok := ext2ValuedOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for ext2 mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}

		switch key {
		case "check":
			if val != "none" && val != "nocheck" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unsupported value '%s' for ext2 mount option 'check' on line %d (only 'none' or 'nocheck' are supported)", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "errors":
			if val != "continue" && val != "remount-ro" && val != "panic" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext2 mount option 'errors' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "resgid", "resuid", "sb":
			if _, err := strconv.Atoi(val); err != nil {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext2 mount option '%s' on line %d, must be an integer", val, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}
	}

	return issues, nil
}

var ext2KnownOptions = map[string]struct{}{
	"acl": {}, "noacl": {}, "bsddf": {}, "minixdf": {},
	"check": {}, "nocheck": {}, "debug": {}, "errors": {},
	"grpid": {}, "bsdgroups": {}, "nogrpid": {}, "sysvgroups": {},
	"grpquota": {}, "usrquota": {}, "quota": {}, "noquota": {},
	"nouid32": {}, "oldalloc": {}, "orlov": {}, "resgid": {},
	"resuid": {}, "sb": {}, "user_xattr": {}, "nouser_xattr": {},

	// Generic ones
	"defaults": {}, "rw": {}, "ro": {}, "noatime": {}, "relatime": {},
	"suid": {}, "nosuid": {}, "exec": {}, "noexec": {}, "auto": {}, "noauto": {},
	"user": {}, "nouser": {}, "sync": {}, "async": {}, "dirsync": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	"nodev": {}, "lazytime": {}, "iversion": {}, "nofail": {},
	"strictatime": {}, "nodiratime": {}, "users": {},

	"_netdev": {}, "dev": {}, "atime": {}, "diratime": {}, "noiversion": {},
	"nolazytime": {}, "owner": {}, "group": {},
}

var ext2ValuedOptions = map[string]struct{}{
	"check": {}, "errors": {}, "resgid": {}, "resuid": {}, "sb": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
}
