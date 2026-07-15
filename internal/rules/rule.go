package rules

import "context"

type Rule interface {
	Name() string
	Dependencies() []string
	Validate(ctx context.Context, entry *FstabEntry) ([]ValidationIssue, error)
}

type IssueLevel string

const (
	IssueLevelWarning IssueLevel = "warning"
	IssueLevelError   IssueLevel = "error"
)

type ValidationIssue struct {
	Message string
	Line    string
	Level   IssueLevel
}

type FstabEntry struct {
	RawLine string
	LineNum int
	Fields  []string
}
