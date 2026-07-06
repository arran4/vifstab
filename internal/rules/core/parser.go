package core

import (
	"bufio"
	"io"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

func ParseFstab(r io.Reader) ([]*rules.FstabEntry, error) {
	scanner := bufio.NewScanner(r)
	var entries []*rules.FstabEntry
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and comments, but still keep them in the structure if we wanted to
		// For now, let's keep everything so we can reconstruct the file

		fields := strings.Fields(line)

		entries = append(entries, &rules.FstabEntry{
			RawLine: line,
			LineNum: lineNum,
			Fields:  fields,
		})
	}

	return entries, scanner.Err()
}
