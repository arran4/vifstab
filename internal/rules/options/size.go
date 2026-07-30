package options

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

// SizeOption validates size strings with optional size suffixes (k, M, G, T, %, etc.).
type SizeOption struct {
	AllowPercentage bool
}

func (s SizeOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	valUpper := strings.ToUpper(val)
	if s.AllowPercentage && strings.HasSuffix(valUpper, "%") {
		numStr := strings.TrimSuffix(valUpper, "%")
		if num, err := strconv.Atoi(numStr); err != nil || num < 0 || num > 100 {
			return []rules.ValidationIssue{
				{
					Message: fmt.Sprintf("Invalid percentage value '%s' for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum),
					Line:    entry.RawLine,
					Level:   rules.IssueLevelError,
				},
			}
		}
		return nil
	}

	numStr := valUpper
	lastChar := valUpper[len(valUpper)-1]
	if lastChar == 'K' || lastChar == 'M' || lastChar == 'G' || lastChar == 'T' || lastChar == 'P' {
		numStr = valUpper[:len(valUpper)-1]
	}

	if num, err := strconv.ParseInt(numStr, 10, 64); err != nil || num < 0 {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value format '%s' for %s on line %d", val, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	return nil
}
