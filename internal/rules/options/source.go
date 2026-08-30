package options

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
)

// SourceOption validates a mount source which could be a path, a device, or a network location.
// Unlike BlockDeviceOption, we don't strictly require existence check unless specified,
// and it can be a file, block device, or something else.
type SourceOption struct {
}

func (s SourceOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing source for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
    // We don't check for existence because a source can be many things (e.g., nfs server path, network location, bind mount path, etc.)
    // We only validate it's present for now.
	return nil
}
