package overlay

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type OverlayRule struct{}

func init() {
	rules.Register(&OverlayRule{})
}

func (r *OverlayRule) Name() string {
	return "overlay"
}

func (r *OverlayRule) Dependencies() []string {
	return []string{}
}

func (r *OverlayRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "overlay" {
		return nil, nil
	}

	var issues []rules.ValidationIssue
	options := rules.SplitOptions(entry.Fields[3])

	hasIndexOff := false
	hasNfsExportOn := false

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		if len(parts) > 1 {
			val = parts[1]
		}

		if _, ok := overlayKnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown overlay mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if val != "" {
			if _, ok := overlayValuedOptions[key]; !ok {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("overlay mount option '%s' does not take a value on line %d", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else {
			if _, ok := overlayValuedOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for overlay mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}

		switch key {
		case "redirect_dir":
			if val != "" && val != "on" && val != "off" && val != "follow" && val != "nofollow" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for overlay mount option 'redirect_dir' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "index":
			if val != "" && val != "on" && val != "off" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for overlay mount option 'index' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			if val == "off" {
				hasIndexOff = true
			}
		case "uuid":
			if val != "" && val != "on" && val != "off" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for overlay mount option 'uuid' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "nfs_export":
			if val != "" && val != "on" && val != "off" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for overlay mount option 'nfs_export' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			if val == "on" {
				hasNfsExportOn = true
			}
		case "xino":
			if val != "" && val != "on" && val != "off" && val != "auto" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for overlay mount option 'xino' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "metacopy":
			if val != "" && val != "on" && val != "off" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Invalid value '%s' for overlay mount option 'metacopy' on line %d", val, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}
	}

	if hasIndexOff && hasNfsExportOn {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("overlay mount options index=off,nfs_export=on are conflicting for a read-write mount on line %d", entry.LineNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		})
	}

	return issues, nil
}

var overlayKnownOptions = map[string]struct{}{
	"lowerdir": {}, "upperdir": {}, "workdir": {}, "userxattr": {},
	"redirect_dir": {}, "index": {}, "uuid": {}, "nfs_export": {},
	"xino": {}, "metacopy": {}, "volatile": {},
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
	"test-opts": {}, "types": {}, "verbose": {}, "version": {}, "x-": {},
}

var overlayValuedOptions = map[string]struct{}{
	"lowerdir": {}, "upperdir": {}, "workdir": {}, "redirect_dir": {}, "index": {},
	"uuid": {}, "nfs_export": {}, "xino": {}, "metacopy": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	"fstab": {}, "map-groups": {}, "map-users": {}, "mkdir": {}, "namespace": {}, "options": {}, "options-mode": {}, "options-source": {},
	"source": {}, "target": {}, "target-prefix": {}, "test-opts": {}, "types": {},
}
