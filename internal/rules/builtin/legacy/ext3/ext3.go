package ext3

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/legacy"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

type Ext3Rule struct{}

func init() {
	rules.Register(&Ext3Rule{})
}

func (r *Ext3Rule) Name() string {
	return "ext3"
}

func (r *Ext3Rule) Dependencies() []string {
	return []string{"linux_fs"}
}

func splitOptions(optsStr string) []string {
	if optsStr == "" {
		return nil
	}
	opts := strings.Split(optsStr, ",")
	return opts
}

func (r *Ext3Rule) Validate(ctx context.Context, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	// ext3 was removed in 4.3, replaced by ext4 handling ext3 format
	issues := legacy.CheckFSVersion(ctx, entry, "ext3", kernel.Ver{}, kernel.Ver{}, false, "ext4 handles ext3-format volumes", false)

	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return issues, nil
	}
	if len(entry.Fields) < 4 {
		return issues, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ext3" {
		return issues, nil
	}

	options := splitOptions(entry.Fields[3])

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		if len(parts) > 1 {
			val = parts[1]
		}

		if _, ok := ext3KnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown ext3 mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if val != "" {
			if _, ok := ext3ValuedOptions[key]; !ok {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("ext3 mount option '%s' does not take a value on line %d", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else {
			if _, ok := ext3ValuedOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for ext3 mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}

		switch key {
		case "check":
			if val != "none" && val != "nocheck" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unsupported value '%s' for ext3 mount option 'check' on line %d (only 'none' or 'nocheck' are supported)", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "errors":
			if val != "continue" && val != "remount-ro" && val != "panic" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext3 mount option 'errors' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "resgid", "resuid", "sb", "journal_dev", "commit":
			if _, err := strconv.Atoi(val); err != nil {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext3 mount option '%s' on line %d, must be an integer", val, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "data":
			if val != "journal" && val != "ordered" && val != "writeback" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext3 mount option 'data' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "data_err":
			if val != "abort" && val != "ignore" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext3 mount option 'data_err' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "barrier":
			if val != "0" && val != "1" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext3 mount option 'barrier' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "jqfmt":
			if val != "vfsold" && val != "vfsv0" && val != "vfsv1" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext3 mount option 'jqfmt' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}
	}

	return issues, nil
}

var ext3KnownOptions = map[string]struct{}{
	// ext2 options
	"acl": {}, "noacl": {}, "bsddf": {}, "minixdf": {},
	"check": {}, "nocheck": {}, "debug": {}, "errors": {},
	"grpid": {}, "bsdgroups": {}, "nogrpid": {}, "sysvgroups": {},
	"grpquota": {}, "usrquota": {}, "quota": {}, "noquota": {},
	"nouid32": {}, "oldalloc": {}, "orlov": {}, "resgid": {},
	"resuid": {}, "sb": {}, "user_xattr": {}, "nouser_xattr": {},

	// ext3 options
	"journal_dev": {}, "journal_path": {}, "norecovery": {}, "noload": {},
	"data": {}, "data_err": {}, "barrier": {}, "commit": {},
	"jqfmt": {}, "usrjquota": {}, "grpjquota": {},

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

var ext3ValuedOptions = map[string]struct{}{
	"check": {}, "errors": {}, "resgid": {}, "resuid": {}, "sb": {},
	"journal_dev": {}, "journal_path": {}, "data": {}, "data_err": {},
	"barrier": {}, "commit": {}, "jqfmt": {}, "usrjquota": {}, "grpjquota": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
}
