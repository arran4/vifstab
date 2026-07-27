package nilfs2

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type Nilfs2Rule struct{}

func init() {
	rules.Register(&Nilfs2Rule{})
}

func (r *Nilfs2Rule) Name() string {
	return "nilfs2"
}

func (r *Nilfs2Rule) Dependencies() []string {
	return []string{}
}

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

func (r *Nilfs2Rule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "nilfs2" {
		return nil, nil
	}

	var issues []rules.ValidationIssue
	options := splitOptions(entry.Fields[3])
	hasRo := false
	hasRw := false

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		if parts[0] == "ro" {
			hasRo = true
		} else if parts[0] == "rw" {
			hasRw = true
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

		if _, ok := nilfs2KnownOptions[key]; !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("nilfs2 mount option '%s' on line %d is unknown", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
			continue
		}

		if hasEquals && val != "" {
			if _, ok := nilfs2ValuedOptions[key]; !ok {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("nilfs2 mount option '%s' on line %d does not take a value", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else if hasEquals && val == "" {
			if _, ok := nilfs2RequireValueOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for nilfs2 mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			} else if _, ok := nilfs2ValuedOptions[key]; !ok {
				if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
					issues = append(issues, rules.ValidationIssue{
						Message: fmt.Sprintf("nilfs2 mount option '%s' on line %d does not take a value", key, entry.LineNum),
						Line:    entry.RawLine,
						Level:   rules.IssueLevelError,
					})
				}
			}
		} else {
			if _, ok := nilfs2RequireValueOptions[key]; ok {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("Missing value for nilfs2 mount option '%s' on line %d", key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		}

		switch key {
		case "errors":
			if val != "continue" && val != "remount-ro" && val != "panic" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("nilfs2 mount option 'errors' on line %d has invalid value '%s'. Valid values are 'continue', 'remount-ro', 'panic'", entry.LineNum, val),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "order":
			if val != "relaxed" && val != "strict" {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("nilfs2 mount option 'order' on line %d has invalid value '%s'. Valid values are 'relaxed', 'strict'", entry.LineNum, val),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "cp":
			if !hasRo || hasRw { // Note: 'ro' must be explicitly specified (or defaults without rw, but explicit rw breaks it)
			    // According to docs: "Snapshot is read-only, so a read-only mount option must be specified together."
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("nilfs2 mount option 'cp' on line %d requires a read-only mount option ('ro')", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				})
			}
		case "norecovery":
			if !hasRo {
				issues = append(issues, rules.ValidationIssue{
					Message: fmt.Sprintf("nilfs2 mount option 'norecovery' on line %d will fail for r/w mounts on an unclean volume. Using read-only ('ro') is recommended.", entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelWarning,
				})
			}
		}
	}

	return issues, nil
}

var nilfs2RequireValueOptions = map[string]struct{}{
	"errors": {}, "cp": {}, "order": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
}

var nilfs2ValuedOptions = map[string]struct{}{
	"errors": {}, "cp": {}, "order": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
}

var nilfs2KnownOptions = map[string]struct{}{
	// nilfs2 specific options
	"barrier": {}, "nobarrier": {}, "errors": {}, "cp": {},
	"order": {}, "norecovery": {}, "discard": {}, "nodiscard": {},

	// Common generic ones
	"defaults": {}, "rw": {}, "ro": {}, "noatime": {}, "relatime": {},
	"suid": {}, "nosuid": {}, "exec": {}, "noexec": {}, "auto": {}, "noauto": {},
	"user": {}, "nouser": {}, "sync": {}, "async": {}, "dirsync": {},
	"context": {}, "defcontext": {}, "rootcontext": {}, "fscontext": {},
	"nodev": {}, "lazytime": {}, "iversion": {}, "nofail": {},
	"strictatime": {}, "nodiratime": {}, "users": {},

	// More generic ones
	"_netdev": {}, "dev": {}, "atime": {}, "diratime": {}, "noiversion": {},
	"nolazytime": {}, "owner": {}, "group": {}, "remount": {}, "bind": {}, "rbind": {},
}
