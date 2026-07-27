package proc

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/kernel"
)

func splitOptions(s string) []string {
	var opts []string
	var current strings.Builder
	inQuote := false
	for _, r := range s {
		if r == '"' {
			inQuote = !inQuote
		}
		if r == ',' && !inQuote {
			opts = append(opts, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		opts = append(opts, current.String())
	}
	return opts
}

type ProcRule struct{}

func init() {
	rules.Register(&ProcRule{})
}

func (r *ProcRule) Name() string {
	return "proc"
}

func (r *ProcRule) Dependencies() []string {
	return []string{}
}

var procKnownOptions = map[string]struct{}{
	"hidepid": {}, "gid": {}, "subset": {},

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

var procValuedOptions = map[string]struct{}{
	"hidepid": {}, "gid": {}, "subset": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
}

func (r *ProcRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "proc" {
		return nil, nil
	}

	var issues []rules.ValidationIssue
	options := splitOptions(entry.Fields[3])


	hostVer := kernel.GetHostKernelVersion(api)

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		if len(parts) > 1 {
			val = parts[1]
		}

		if _, ok := procKnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Unknown proc mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if val != "" {
			if _, ok := procValuedOptions[key]; !ok {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("proc mount option '%s' does not take a value on line %d", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else {
			if _, ok := procValuedOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for proc mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}

		if key == "hidepid" {
			if hostVer.LessThan(3, 3, 0) {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("proc mount option 'hidepid' is only supported on Linux 3.3+ (detected %s)", hostVer),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		} else if key == "gid" {
			if hostVer.LessThan(3, 3, 0) {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("proc mount option 'gid' is only supported on Linux 3.3+ (detected %s)", hostVer),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		} else if key == "subset" {
			if hostVer.LessThan(5, 8, 0) {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("proc mount option 'subset' is only supported on Linux 5.8+ (detected %s)", hostVer),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			if val != "pid" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("proc mount option 'subset' expects 'pid', got '%s'", val),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}
	}

	return issues, nil
}
