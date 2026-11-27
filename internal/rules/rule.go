package rules

import "context"

type Rule interface {
	Name() string
	Dependencies() []string
	Validate(ctx context.Context, api API, entry *FstabEntry) ([]ValidationIssue, error)
}

// FileRule is an optional interface for rules that validate file-level properties
// once per fstab file, including empty files with zero entries.
type FileRule interface {
	Rule
	ValidateFile(ctx context.Context, api API) ([]ValidationIssue, error)
}

type IssueLevel string

const (
	IssueLevelWarning  IssueLevel = "warning"
	IssueLevelSecurity IssueLevel = "security"
	IssueLevelError    IssueLevel = "error"
	IssueLevelInfo     IssueLevel = "info"
)

func IsBlockingIssue(issue ValidationIssue) bool {
	return issue.Level == IssueLevelWarning || issue.Level == IssueLevelError || issue.Level == IssueLevelSecurity
}

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
