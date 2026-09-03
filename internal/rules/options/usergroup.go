package options

import (
	"context"
	"fmt"
	"strconv"

	"github.com/arran4/vifstab/internal/rules"
)

// UIDOption validates user IDs.
type UIDOption struct {
	CustomUserNotFound string
	Level              rules.IssueLevel
}

func (u UIDOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	num, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be an integer", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	switch api.CheckUserExists(int(num)) {
	case rules.DoesNotExist:
		msg := fmt.Sprintf("User with uid %d does not exist", num)
		if u.CustomUserNotFound != "" {
			msg = fmt.Sprintf(u.CustomUserNotFound, val, entry.LineNum)
		}
		level := u.Level
		if level == "" {
			level = rules.IssueLevelError
		}
		return []rules.ValidationIssue{
			{
				Message: msg,
				Line:    entry.RawLine,
				Level:   level,
			},
		}
	case rules.PermissionDenied:
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Permission denied while checking if user %d exists", num),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelWarning,
			},
		}
	case rules.Unknown:
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Unable to determine if user %d exists", num),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelInfo,
			},
		}
	}
	return nil
}

// GIDOption validates group IDs.
type GIDOption struct {
	CustomGroupNotFound string
	Level               rules.IssueLevel
}

func (g GIDOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	num, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be an integer", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	switch api.CheckGroupExists(int(num)) {
	case rules.DoesNotExist:
		msg := fmt.Sprintf("Group with gid %d does not exist", num)
		if g.CustomGroupNotFound != "" {
			msg = fmt.Sprintf(g.CustomGroupNotFound, val, entry.LineNum)
		}
		level := g.Level
		if level == "" {
			level = rules.IssueLevelError
		}
		return []rules.ValidationIssue{
			{
				Message: msg,
				Line:    entry.RawLine,
				Level:   level,
			},
		}
	case rules.PermissionDenied:
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Permission denied while checking if group %d exists", num),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelWarning,
			},
		}
	case rules.Unknown:
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Unable to determine if group %d exists", num),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelInfo,
			},
		}
	}
	return nil
}

// OptionalUIDOption allows no value or a UID integer value.
type OptionalUIDOption struct{}

func (o OptionalUIDOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return nil
	}
	return (UIDOption{}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
}

// OptionalGIDOption allows no value or a GID integer value.
type OptionalGIDOption struct{}

func (o OptionalGIDOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return nil
	}
	return (GIDOption{}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
}

// UIDOrEnumOption validates that an option is either a specific set of strings, or a valid user ID.
type UIDOrEnumOption struct {
	AllowedEnum map[string]bool
}

func (o UIDOrEnumOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	if o.AllowedEnum != nil && o.AllowedEnum[val] {
		return nil
	}

	// Try validating as UID
	issues := (UIDOption{}).ValidateOption(ctx, api, entry, fsName, key, val, hasVal, allOpts)
	if len(issues) > 0 {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be an allowed string or a valid UID", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}
