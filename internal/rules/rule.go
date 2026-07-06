package rules

import "context"

type Rule interface {
	Name() string
	Dependencies() []string
	Validate(ctx context.Context, entry *FstabEntry) ([]Suggestion, error)
}

type SuggestionLevel string

const (
	SuggestionLevelWarning SuggestionLevel = "warning"
	SuggestionLevelError   SuggestionLevel = "error"
)

type Suggestion struct {
	Message string
	Line    string
	Level   SuggestionLevel
}

type FstabEntry struct {
	RawLine string
	LineNum int
	Fields  []string
}
