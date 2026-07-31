package options

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
)

// FileOption validates a file path, ensuring it exists if CheckExistence is true.
type FileOption struct {
	CheckExistence bool
	// If CheckExistence is true but we lack permissions, we can log a warning instead of error.
	// But api.CheckFileExists currently only returns a bool. So this behaves identically for now.
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
		if !api.CheckFileExists(decodedVal) {
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
