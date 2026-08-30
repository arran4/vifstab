package options

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/arran4/vifstab/internal/rules"
)

// PathOption validates a path, optionally ensuring it is an absolute path.
type PathOption struct {
	RequireAbsolute bool
}

func (p PathOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing path for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	decodedVal := unescapeFstab(val)
	if p.RequireAbsolute && !filepath.IsAbs(decodedVal) {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Path '%s' for %s mount option '%s' must be an absolute path on line %d", decodedVal, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	return nil
}
