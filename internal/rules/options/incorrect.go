package options

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
)

// IncorrectOption warns the user that this option is incorrectly used here.
type IncorrectOption struct {
	Reason string
}

func (o IncorrectOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	return []rules.ValidationIssue{
		{
			Message: fmt.Sprintf("Mount option '%s' is not valid for %s: %s on line %d", key, fsName, o.Reason, entry.LineNum),
			Line:    entry.RawLine,
			Level:   rules.IssueLevelError,
		},
	}
}
