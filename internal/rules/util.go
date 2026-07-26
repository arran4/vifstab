package rules

import "strings"

// SplitOptions splits a comma-separated options string into a slice of options,
// respecting double quotes.
func SplitOptions(optsStr string) []string {
	if optsStr == "" {
		return nil
	}
	var opts []string
	var current strings.Builder
	inQuote := false
	for _, r := range optsStr {
		if r == '"' {
			inQuote = !inQuote
		}
		if r == ',' && !inQuote {
			opts = append(opts, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		opts = append(opts, current.String())
	}
	return opts
}
