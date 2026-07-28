package ext4

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Ext4Rule struct{}

func init() {
	rules.Register(&Ext4Rule{})
}

func (r *Ext4Rule) Name() string {
	return "ext4"
}

func (r *Ext4Rule) Dependencies() []string {
	return []string{}
}

func (r *Ext4Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "ext4" {
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

		if _, ok := ext4KnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown ext4 mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if val != "" {
			if _, ok := ext4ValuedOptions[key]; !ok {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("ext4 mount option '%s' does not take a value on line %d", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else {
			if _, ok := ext4ValuedOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for ext4 mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}

		switch key {
		case "check":
			if val != "none" && val != "nocheck" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unsupported value '%s' for ext4 mount option 'check' on line %d (only 'none' or 'nocheck' are supported)", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "errors":
			if val != "continue" && val != "remount-ro" && val != "panic" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext4 mount option 'errors' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "resgid", "resuid", "sb", "journal_dev", "commit", "inode_readahead_blks", "stripe", "max_batch_time", "min_batch_time", "journal_ioprio", "init_itable", "max_dir_size_kb":
			if _, err := strconv.Atoi(val); err != nil {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext4 mount option '%s' on line %d, must be an integer", val, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			} else {
				num, _ := strconv.Atoi(val)
				if key == "journal_ioprio" {
					if num < 0 || num > 7 {
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Invalid value '%d' for ext4 mount option 'journal_ioprio' on line %d, must be between 0 and 7", num, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					}
				}
				if key == "inode_readahead_blks" {
					if num&(num-1) != 0 || num == 0 { // Must be power of 2
						issues = append(issues, rules.ValidationIssue{
							Message: fmt.Sprintf("Invalid value '%d' for ext4 mount option 'inode_readahead_blks' on line %d, must be a power of 2", num, entry.LineNum),
							Line:    entry.RawLine,
							Level:   rules.IssueLevelError,
						})
					}
				}
			}
		case "data":
			if val != "journal" && val != "ordered" && val != "writeback" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext4 mount option 'data' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "data_err":
			if val != "abort" && val != "ignore" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext4 mount option 'data_err' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "barrier":
			if val != "0" && val != "1" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext4 mount option 'barrier' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "jqfmt":
			if val != "vfsold" && val != "vfsv0" && val != "vfsv1" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for ext4 mount option 'jqfmt' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}
	}

	return issues, nil
}

var ext4KnownOptions = map[string]struct{}{
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

	// ext4 options
	"journal_checksum": {}, "nojournal_checksum": {}, "journal_async_commit": {},
	"nobarrier": {}, "inode_readahead_blks": {}, "stripe": {}, "delalloc": {},
	"nodelalloc": {}, "max_batch_time": {}, "min_batch_time": {}, "journal_ioprio": {},
	"abort": {}, "auto_da_alloc": {}, "noauto_da_alloc": {}, "noinit_itable": {},
	"init_itable": {}, "discard": {}, "nodiscard": {}, "block_validity": {},
	"noblock_validity": {}, "dioread_lock": {}, "dioread_nolock": {}, "max_dir_size_kb": {},
	"i_version": {}, "nombcache": {}, "prjquota": {},

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

var ext4ValuedOptions = map[string]struct{}{
	"check": {}, "errors": {}, "resgid": {}, "resuid": {}, "sb": {},
	"journal_dev": {}, "journal_path": {}, "data": {}, "data_err": {},
	"barrier": {}, "commit": {}, "jqfmt": {}, "usrjquota": {}, "grpjquota": {},
	"inode_readahead_blks": {}, "stripe": {}, "max_batch_time": {}, "min_batch_time": {},
	"journal_ioprio": {}, "init_itable": {}, "max_dir_size_kb": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	// Generic valued options added automatically
	"fstab": {}, "map-groups": {}, "map-users": {}, "mkdir": {}, "namespace": {}, "options": {}, "options-mode": {}, "options-source": {},
	"source": {}, "target": {}, "target-prefix": {}, "test-opts": {}, "types": {}, "uuid": {},
}
