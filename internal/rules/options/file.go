package options

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
)

// FileOption validates a file path, ensuring it exists if CheckExistence is true.
type FileOption struct {
	CheckExistence              bool
	IgnoreFailureIfNoPermission bool
}

func (f FileOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing file path for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	if f.CheckExistence {
		decodedVal := unescapeFstab(val)
		existsResult := api.CheckFileExists(decodedVal)

		if existsResult != rules.FileExists {
			if existsResult == rules.Unknown {
				return []rules.ValidationIssue{{
					Message: fmt.Sprintf("Unable to determine existence of file '%s' for %s mount option '%s' on line %d", decodedVal, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelInfo,
				}}
			}
			if existsResult == rules.FilePermissionDenied && f.IgnoreFailureIfNoPermission {
				// We lack permission to check, but we are allowed to ignore it.
				// However, if we're ignoring the failure entirely, we shouldn't emit an error.
				return nil
			}
			if existsResult == rules.FilePermissionDenied {
				return []rules.ValidationIssue{{
					Message: fmt.Sprintf("Permission denied while checking file '%s' for %s mount option '%s' on line %d", decodedVal, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelInfo,
				}}
			}
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("File '%s' specified in %s mount option '%s' does not exist on line %d", decodedVal, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
	}
	return nil
}
