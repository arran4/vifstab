package format

import (
	"fmt"
	"io"
	"strings"

	"github.com/arran4/vifstab/internal/rules/core"
)

// Format reads fstab entries from r and writes a column-aligned version to w.
func Format(r io.Reader, w io.Writer) error {
	entries, err := core.ParseFstab(r)
	if err != nil {
		return fmt.Errorf("failed to parse fstab: %w", err)
	}

	// Calculate max column widths
	colWidths := make(map[int]int)
	for _, entry := range entries {
		line := strings.TrimSpace(entry.RawLine)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		for i, field := range entry.Fields {
			if len(field) > colWidths[i] {
				colWidths[i] = len(field)
			}
		}
	}

	for _, entry := range entries {
		line := strings.TrimSpace(entry.RawLine)
		if len(line) == 0 {
			if _, err := fmt.Fprintln(w, ""); err != nil {
				return err
			}
			continue
		}
		if line[0] == '#' {
			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
			continue
		}

		var formattedLine []string
		for i, field := range entry.Fields {
			// Do not pad the last column
			if i == len(entry.Fields)-1 {
				formattedLine = append(formattedLine, field)
			} else {
				formattedLine = append(formattedLine, fmt.Sprintf("%-*s", colWidths[i], field))
			}
		}
		if _, err := fmt.Fprintln(w, strings.Join(formattedLine, "  ")); err != nil {
			return err
		}
	}
	return nil
}
