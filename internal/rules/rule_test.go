package rules

import (
	"testing"
)

func TestIsBlockingIssue(t *testing.T) {
	tests := []struct {
		name     string
		issue    ValidationIssue
		expected bool
	}{
		{
			name: "warning level is blocking",
			issue: ValidationIssue{
				Level: IssueLevelWarning,
			},
			expected: true,
		},
		{
			name: "error level is blocking",
			issue: ValidationIssue{
				Level: IssueLevelError,
			},
			expected: true,
		},
		{
			name: "security level is blocking",
			issue: ValidationIssue{
				Level: IssueLevelSecurity,
			},
			expected: true,
		},
		{
			name: "info level is non-blocking",
			issue: ValidationIssue{
				Level: IssueLevelInfo,
			},
			expected: false,
		},
		{
			name: "empty issue level is non-blocking",
			issue: ValidationIssue{
				Level: "",
			},
			expected: false,
		},
		{
			name: "unknown issue level is non-blocking",
			issue: ValidationIssue{
				Level: IssueLevel("unknown"),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBlockingIssue(tt.issue)
			if got != tt.expected {
				t.Errorf("IsBlockingIssue(%v) = %v; want %v", tt.issue.Level, got, tt.expected)
			}
		})
	}
}
