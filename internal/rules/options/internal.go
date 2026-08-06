package options

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
)

// InternalOption warns the user that this option is an internal kernel flag and should not be used in fstab.
type InternalOption struct{}

func (o InternalOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	return []rules.ValidationIssue{
		{
			Message: fmt.Sprintf("Mount option '%s' is an internal kernel flag and not valid for %s in fstab on line %d", key, fsName, entry.LineNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		},
	}
}
