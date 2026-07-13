package editor

import (
	"fmt"
	"strings"
	"unicode"
)

// SplitEditorCommand splits a shell command string into parts, respecting single and double quotes.
// This is a minimal implementation to avoid external dependencies like shlex.
func SplitEditorCommand(s string) ([]string, error) {
	var parts []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for _, r := range s {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		if r == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}

		if r == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if unicode.IsSpace(r) && !inSingleQuote && !inDoubleQuote {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(r)
	}

	if escaped {
		return nil, fmt.Errorf("unterminated escape sequence at end of string")
	}
	if inSingleQuote {
		return nil, fmt.Errorf("unterminated single quote")
	}
	if inDoubleQuote {
		return nil, fmt.Errorf("unterminated double quote")
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts, nil
}
