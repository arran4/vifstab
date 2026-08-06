package format

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/arran4/vifstab/internal/rules/core"
)

// Format reads fstab entries from r and writes a column-aligned version to w.
func Format(r io.Reader, w io.Writer) error {
	entries, err := core.ParseFstab(r)
	if err != nil {
		return fmt.Errorf("failed to parse fstab: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.DiscardEmptyColumns)
	for _, entry := range entries {
		line := strings.TrimSpace(entry.RawLine)
		if len(line) == 0 {
			tw.Flush()
			fmt.Fprintln(w, "")
			continue
		}
		if line[0] == '#' {
			tw.Flush()
			fmt.Fprintln(w, line)
			continue
		}

		fmt.Fprintln(tw, strings.Join(entry.Fields, "\t"))
	}
	return tw.Flush()
}
