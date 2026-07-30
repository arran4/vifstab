package options

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// BoolSet represents a bitmask of allowed boolean keyword representations.
type BoolSet uint8

const (
	BoolSetNumeric  BoolSet = 1 << iota // "0", "1"
	BoolSetYesNo                        // "yes", "no"
	BoolSetTrueFalse                    // "true", "false"
	BoolSetOnOff                        // "on", "off"

	BoolSetStandard = BoolSetNumeric | BoolSetYesNo | BoolSetTrueFalse // 0, 1, yes, no, true, false
	BoolSetAll      = BoolSetStandard | BoolSetOnOff                  // 0, 1, yes, no, true, false, on, off
)

// BoolOption accepts boolean values configured via enabled BoolSet flags.
type BoolOption struct {
	Sets BoolSet // Enabled value sets. If 0, defaults to BoolSetStandard.
}

func (b BoolOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return nil
	}

	sets := b.Sets
	if sets == 0 {
		sets = BoolSetStandard
	}

	allowed := make(map[string]bool)
	var allowedDescriptions []string

	if sets&BoolSetNumeric != 0 {
		allowed["0"] = true
		allowed["1"] = true
		allowedDescriptions = append(allowedDescriptions, "0", "1")
	}
	if sets&BoolSetYesNo != 0 {
		allowed["yes"] = true
		allowed["no"] = true
		allowedDescriptions = append(allowedDescriptions, "yes", "no")
	}
	if sets&BoolSetTrueFalse != 0 {
		allowed["true"] = true
		allowed["false"] = true
		allowedDescriptions = append(allowedDescriptions, "true", "false")
	}
	if sets&BoolSetOnOff != 0 {
		allowed["on"] = true
		allowed["off"] = true
		allowedDescriptions = append(allowedDescriptions, "on", "off")
	}

	if !allowed[val] {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d, must be a boolean (%s)", val, fsName, key, entry.LineNum, strings.Join(allowedDescriptions, ", ")),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	return nil
}
