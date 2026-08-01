package umsdos

import (
	"context"
	"fmt"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/local/fat"
)

type UmsdosRule struct{}

func init() {
	rules.Register(&UmsdosRule{})
}

func (r *UmsdosRule) Name() string {
	return "umsdos"
}

func (r *UmsdosRule) Dependencies() []string {
	return []string{}
}

func (r *UmsdosRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	if len(entry.Fields) == 0 || strings.HasPrefix(strings.TrimSpace(entry.RawLine), "#") {
		return nil, nil
	}
	if len(entry.Fields) < 4 {
		return nil, nil
	}

	fsType := entry.Fields[2]
	if fsType != "umsdos" {
		return nil, nil
	}

	var issues []rules.ValidationIssue
	options := rules.SplitOptions(entry.Fields[3])

	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		key := parts[0]
		if key == "dotsOK" {
			issues = append(issues, rules.ValidationIssue{
				Message: fmt.Sprintf("umsdos explicitly kills dotsOK mount option on line %d", entry.LineNum),
				Line:    entry.RawLine,
				Level:   rules.IssueLevelError,
			})
		}
	}

	// Delegate to fat options
	fatRule := &fat.FatRule{}
	mockEntry := *entry
	mockEntry.Fields = append([]string{}, entry.Fields...)
	mockEntry.Fields[2] = "fat"

	fatIssues, err := fatRule.Validate(ctx, api, &mockEntry)
	if err != nil {
		return issues, err
	}

	// Adjust messages to say 'umsdos' instead of 'fat'
	for _, issue := range fatIssues {
		issue.Message = strings.ReplaceAll(issue.Message, "fat mount option", "umsdos mount option")
		issues = append(issues, issue)
	}

	return issues, nil
}
