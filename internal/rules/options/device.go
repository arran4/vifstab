package options

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// BlockDeviceOption validates block device paths (e.g. /dev/sda1, UUID=..., etc.).
type BlockDeviceOption struct {
	CheckExistence bool
}

func unescapeFstab(s string) string {
	s = strings.ReplaceAll(s, "\\040", " ")
	s = strings.ReplaceAll(s, "\\011", "\t")
	s = strings.ReplaceAll(s, "\\012", "\n")
	s = strings.ReplaceAll(s, "\\134", "\\")
	return s
}

func (b BlockDeviceOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing device for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	if b.CheckExistence {
		decodedVal := unescapeFstab(val)
		if !api.CheckBlockDeviceExists(decodedVal) {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Device '%s' specified in %s mount option %s does not exist on line %d", decodedVal, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
	}
	return nil
}
