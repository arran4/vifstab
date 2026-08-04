package gocryptfs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type ForceOwnerOption struct{}

func (f ForceOwnerOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]options.OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	parts := strings.Split(val, ":")
	if len(parts) != 2 {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be 'uid:gid'", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	var issues []rules.ValidationIssue

	uidStr, gidStr := parts[0], parts[1]

	uidNum, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("Invalid uid value '%s' in %s mount option '%s' on line %d, must be an integer", uidStr, fsName, key, entry.LineNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		})
	} else if !api.CheckUserExists(int(uidNum)) {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("User with uid %d does not exist", uidNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		})
	}

	gidNum, err := strconv.ParseInt(gidStr, 10, 64)
	if err != nil {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("Invalid gid value '%s' in %s mount option '%s' on line %d, must be an integer", gidStr, fsName, key, entry.LineNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		})
	} else if !api.CheckGroupExists(int(gidNum)) {
		issues = append(issues, rules.ValidationIssue{
			Message: fmt.Sprintf("Group with gid %d does not exist", gidNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		})
	}

	return issues
}
