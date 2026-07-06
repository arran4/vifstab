package rules

import "context"

type Rule interface {
	Name() string
	Dependencies() []string
	Validate(ctx context.Context, entry *FstabEntry) []Suggestion
}

type Suggestion struct {
	Message string
	Line    string
}

type FstabEntry struct {
	RawLine string
	LineNum int
	Fields  []string
}
