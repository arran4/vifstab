package rules

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// OptionValidator defines the interface for validating mount options.
type OptionValidator interface {
	ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue
}

// OptionValidatorFunc allows plain functions to satisfy OptionValidator.
type OptionValidatorFunc func(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue

func (f OptionValidatorFunc) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	return f(ctx, api, entry, fsName, key, val, hasVal)
}

// NoValueOption validates that an option does not accept a value.
type NoValueOption struct{}

func (NoValueOption) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	if val != "" {
		if !strings.HasPrefix(key, "X-mount.") && !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") {
			return []ValidationIssue{
				{
					Message: fmt.Sprintf("%s mount option '%s' does not take a value on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   IssueLevelError,
				},
			}
		}
	}
	return nil
}

// StringOption validates that an option requires a string value.
type StringOption struct{}

func (StringOption) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	if !hasVal || val == "" {
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	return nil
}

// OptionalStringOption allows either no value or any string value.
type OptionalStringOption struct{}

func (OptionalStringOption) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	return nil
}

// IntOption validates that an option has an integer value.
type IntOption struct {
	Min *int64
	Max *int64
}

func (i IntOption) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	if !hasVal || val == "" {
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	num, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be an integer", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	if i.Min != nil && num < *i.Min {
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Value '%s' for %s mount option '%s' on line %d is less than minimum %d", val, fsName, key, entry.LineNum, *i.Min),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	if i.Max != nil && num > *i.Max {
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Value '%s' for %s mount option '%s' on line %d is greater than maximum %d", val, fsName, key, entry.LineNum, *i.Max),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	return nil
}

// OctalOption validates octal permissions value.
type OctalOption struct {
	CustomErrFormat string // e.g. "must be an octal value" instead of "must be an octal integer"
}

func (o OctalOption) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	if !hasVal || val == "" {
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	if _, err := strconv.ParseUint(val, 8, 32); err != nil {
		msgType := "must be an octal integer"
		if o.CustomErrFormat != "" {
			msgType = o.CustomErrFormat
		}
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, %s", val, fsName, key, entry.LineNum, msgType),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	return nil
}

// UIDOption validates user IDs.
type UIDOption struct {
	CustomUserNotFound string
	Level              IssueLevel
}

func (u UIDOption) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	if !hasVal || val == "" {
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	num, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be an integer", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	if !api.CheckUserExists(int(num)) {
		msg := fmt.Sprintf("User with uid %d does not exist", num)
		if u.CustomUserNotFound != "" {
			msg = fmt.Sprintf(u.CustomUserNotFound, val, entry.LineNum)
		}
		level := u.Level
		if level == "" {
			level = IssueLevelError
		}
		return []ValidationIssue{
			{
				Message: msg,
				Line:    entry.RawLine,
				Level:   level,
			},
		}
	}
	return nil
}

// GIDOption validates group IDs.
type GIDOption struct {
	CustomGroupNotFound string
	Level               IssueLevel
}

func (g GIDOption) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	if !hasVal || val == "" {
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	num, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be an integer", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	if !api.CheckGroupExists(int(num)) {
		msg := fmt.Sprintf("Group with gid %d does not exist", num)
		if g.CustomGroupNotFound != "" {
			msg = fmt.Sprintf(g.CustomGroupNotFound, val, entry.LineNum)
		}
		level := g.Level
		if level == "" {
			level = IssueLevelError
		}
		return []ValidationIssue{
			{
				Message: msg,
				Line:    entry.RawLine,
				Level:   level,
			},
		}
	}
	return nil
}

// OptionalUIDOption allows no value or a UID integer value.
type OptionalUIDOption struct{}

func (o OptionalUIDOption) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	if !hasVal || val == "" {
		return nil
	}
	return (UIDOption{}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal)
}

// OptionalGIDOption allows no value or a GID integer value.
type OptionalGIDOption struct{}

func (o OptionalGIDOption) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	if !hasVal || val == "" {
		return nil
	}
	return (GIDOption{}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal)
}


// BoolOption accepts optional boolean values (0, 1, yes, no, true, false).
type BoolOption struct{}

func (BoolOption) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	if hasVal && val != "" {
		if val != "0" && val != "1" && val != "yes" && val != "no" && val != "true" && val != "false" {
			return []ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be a boolean (0, 1, yes, no, true, false)", val, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   IssueLevelError,
				},
			}
		}
	}
	return nil
}

// EnumOption validates option values against a fixed list of allowed strings.
type EnumOption struct {
	Allowed map[string]bool
	CustomErrMsg string
}

func (e EnumOption) ValidateOption(ctx context.Context, api API, entry *FstabEntry, fsName string, key string, val string, hasVal bool) []ValidationIssue {
	if !hasVal || val == "" {
		return []ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	if !e.Allowed[val] {
		msg := e.CustomErrMsg
		if msg == "" {
			msg = fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum)
		} else {
			msg = fmt.Sprintf(msg, val, entry.LineNum)
		}
		return []ValidationIssue{
			{
				Message: msg,
				Line:    entry.RawLine,
				Level:   IssueLevelError,
			},
		}
	}
	return nil
}

// MergeOptions merges multiple OptionValidator maps into a single map.
func MergeOptions(maps ...map[string]OptionValidator) map[string]OptionValidator {
	res := make(map[string]OptionValidator)
	for _, m := range maps {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

// CommonOptions returns standard mount options common to Linux filesystems.
func CommonOptions() map[string]OptionValidator {
	opts := map[string]OptionValidator{
		// Flags / No value options
		"defaults": NoValueOption{}, "rw": NoValueOption{}, "ro": NoValueOption{},
		"noatime": NoValueOption{}, "relatime": NoValueOption{}, "suid": NoValueOption{},
		"nosuid": NoValueOption{}, "exec": NoValueOption{}, "noexec": NoValueOption{},
		"auto": NoValueOption{}, "noauto": NoValueOption{}, "user": NoValueOption{},
		"nouser": NoValueOption{}, "sync": NoValueOption{}, "async": NoValueOption{},
		"dirsync": NoValueOption{}, "nodev": NoValueOption{}, "lazytime": NoValueOption{},
		"iversion": NoValueOption{}, "nofail": NoValueOption{}, "strictatime": NoValueOption{},
		"nodiratime": NoValueOption{}, "users": NoValueOption{}, "_netdev": NoValueOption{},
		"dev": NoValueOption{}, "atime": NoValueOption{}, "diratime": NoValueOption{},
		"noiversion": NoValueOption{}, "nolazytime": NoValueOption{}, "owner": NoValueOption{},
		"group": NoValueOption{},
		"beneath": NoValueOption{}, "bind": NoValueOption{}, "exclusive": NoValueOption{},
		"fake": NoValueOption{}, "fork": NoValueOption{}, "help": NoValueOption{},
		"internal-only": NoValueOption{}, "loud": NoValueOption{}, "make-private": NoValueOption{},
		"make-rprivate": NoValueOption{}, "make-rshared": NoValueOption{}, "make-rslave": NoValueOption{},
		"make-runbindable": NoValueOption{}, "make-shared": NoValueOption{}, "make-slave": NoValueOption{},
		"make-unbindable": NoValueOption{}, "no-canonicalize": NoValueOption{}, "no-mtab": NoValueOption{},
		"nomand": NoValueOption{}, "nosymfollow": NoValueOption{}, "onlyonce": NoValueOption{},
		"options-source-force": NoValueOption{}, "rbind": NoValueOption{}, "read-only": NoValueOption{},
		"read-write": NoValueOption{}, "remount": NoValueOption{}, "show-labels": NoValueOption{},
		"silent": NoValueOption{}, "verbose": NoValueOption{}, "version": NoValueOption{},
		"X-": NoValueOption{}, "x-": NoValueOption{},

		// Valued generic options
		"context": StringOption{}, "defcontext": StringOption{}, "rootcontext": StringOption{},
		"fscontext": StringOption{}, "fstab": StringOption{}, "map-groups": StringOption{},
		"map-users": StringOption{}, "mkdir": StringOption{}, "namespace": StringOption{},
		"options": StringOption{}, "options-mode": StringOption{}, "options-source": StringOption{},
		"source": StringOption{}, "target": StringOption{}, "target-prefix": StringOption{},
		"test-opts": StringOption{}, "types": StringOption{}, "uuid": StringOption{},
	}
	return opts
}

// ValidateOptionsMap iterates through options in entry.Fields[3] and validates them against optsMap.
func ValidateOptionsMap(ctx context.Context, api API, entry *FstabEntry, fsName string, optsMap map[string]OptionValidator) []ValidationIssue {
	if len(entry.Fields) < 4 {
		return nil
	}

	var issues []ValidationIssue
	options := SplitOptions(entry.Fields[3])

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		var val string
		hasVal := len(parts) > 1
		if hasVal {
			val = parts[1]
		}

		validator, ok := optsMap[key]
		if !ok {
			if !strings.HasPrefix(key, "x-systemd.") && !strings.HasPrefix(key, "x-") && !strings.HasPrefix(key, "X-") {
				issues = append(issues, ValidationIssue{
					Message: fmt.Sprintf("Unknown %s mount option '%s' on line %d", fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   IssueLevelError,
				})
			}
			continue
		}

		if validator != nil {
			issues = append(issues, validator.ValidateOption(ctx, api, entry, fsName, key, val, hasVal)...)
		}
	}

	return issues
}
