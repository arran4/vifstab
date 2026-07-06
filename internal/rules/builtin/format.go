package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

type FormatRule struct{}

func init() {
	rules.Register(&FormatRule{})
}

func (r *FormatRule) Name() string {
	return "format"
}

func (r *FormatRule) Dependencies() []string {
	return nil
}

func (r *FormatRule) Validate(ctx context.Context, entry *rules.FstabEntry) []rules.Suggestion {
	if len(entry.Fields) == 0 {
		return nil
	}
	if strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil
	}

	if len(entry.Fields) < 6 {
		return []rules.Suggestion{
			{
				Message: fmt.Sprintf("Line %d has less than 6 fields", entry.LineNum),
				Line:    entry.RawLine,
			},
		}
	}
	return nil
}
