package rules

import "context"

type Rule interface {
	Name() string
	Dependencies() []string
	Validate(ctx context.Context, api API, entry *FstabEntry) ([]ValidationIssue, error)
}

type IssueLevel string

const (
	IssueLevelWarning  IssueLevel = "warning"
	IssueLevelSecurity IssueLevel = "security"
	IssueLevelError    IssueLevel = "error"
	IssueLevelInfo     IssueLevel = "info"
)

type ValidationIssue struct {
	Message  string
	Line     string
	Level    IssueLevel
	Severity int
}

type FstabEntry struct {
	RawLine string
	LineNum int
	Fields  []string
}
