package options

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

//go:embed charsets.txt
var charsetsData string

var validCharsets map[string]bool

func init() {
	validCharsets = make(map[string]bool)
	data := charsetsData
	for len(data) > 0 {
		var line string
		if idx := strings.IndexByte(data, '\n'); idx >= 0 {
			line, data = data[:idx], data[idx+1:]
		} else {
			line, data = data, ""
		}
		line = strings.TrimSpace(line)
		if line != "" {
			validCharsets[strings.ToLower(line)] = true
		}
	}
}

// IOCharsetOption validates iocharset values (NLS translation tables).
type IOCharsetOption struct{}

func (IOCharsetOption) ValidateOption(ctx context.Context, api rules.API, entry *rules.FstabEntry, fsName string, key string, val string, hasVal bool, allOpts map[string]OptionState) []rules.ValidationIssue {
	if !hasVal || val == "" {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Missing value for %s mount option '%s' on line %d", fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}

	if !validCharsets[strings.ToLower(val)] {
		return []rules.ValidationIssue{
			{
				Message: fmt.Sprintf("Invalid value '%s' for %s mount option '%s' on line %d", val, fsName, key, entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			},
		}
	}
	return nil
}
