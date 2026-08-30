package options

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
)

// DirOption validates a directory path, ensuring it exists if CheckExistence is true.
type DirOption struct {
	CheckExistence              bool
	IgnoreFailureIfNoPermission bool
}

func (d DirOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing directory path for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	if d.CheckExistence {
		decodedVal := unescapeFstab(val)
		existsResult := api.CheckFileExists(decodedVal)

		if existsResult != rules.FileExists {
			if existsResult == rules.FilePermissionDenied && d.IgnoreFailureIfNoPermission {
				// We lack permission to check, but we are allowed to ignore it.
				// However, if we're ignoring the failure entirely, we shouldn't emit an error.
				return nil
			}
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Directory '%s' specified in %s mount option '%s' does not exist on line %d", decodedVal, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
	}
	return nil
}
