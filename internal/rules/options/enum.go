package options

import (
	"context"
	"fmt"

	"github.com/arran4/vifstab/internal/rules"
)

// EnumOption validates option values against a fixed list of allowed strings.
type EnumOption struct {
	Allowed      map[string]bool
	CustomErrMsg string
}

func (e EnumOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	if !e.Allowed[val] {
		msg := e.CustomErrMsg
		if msg == "" {
			msg = fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum)
		} else {
			// Count %s to see if it wants the fsName argument too
			// CustomErrMsg either uses format (val, line) or (val, fsName, line)
			countS := 0
			for i := 0; i < len(msg)-1; i++ {
				if msg[i] == '%' && msg[i+1] == 's' {
					countS++
				}
			}

			if countS >= 2 {
				msg = fmt.Sprintf(msg, val, fsName, entry.LineNum)
			} else {
				msg = fmt.Sprintf(msg, val, entry.LineNum)
			}
		}
		return []rules.ValidationIssue{
			{
				Message: msg,
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}
