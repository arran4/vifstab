package msdos

import (
	"context"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin/local/fat"
)

type MsdosRule struct{}

func init() {
	rules.Register(&MsdosRule{})
}

func (r *MsdosRule) Name() string {
	return "msdos"
}

func (r *MsdosRule) Dependencies() []string {
	return []string{}
}

func (r *MsdosRule) Validate(ctx context.Context, api rules.API, entry *rules.FstabEntry) ([]rules.ValidationIssue, error) {
	// msdos uses same options as fat
	fatRule := &fat.FatRule{}

	// Create a copy of the entry but with fsType as fat to pass fat validation
	if len(entry.Fields) >= 3 && entry.Fields[2] == "msdos" {
		mockEntry := *entry
		mockEntry.Fields = append([]string{}, entry.Fields...)
		mockEntry.Fields[2] = "fat"

		fatIssues, err := fatRule.Validate(ctx, api, &mockEntry)
		if err != nil {
			return nil, err
		}

		var issues []rules.ValidationIssue
		for _, issue := range fatIssues {
			issue.Message = strings.ReplaceAll(issue.Message, "fat mount option", "msdos mount option")
			issues = append(issues, issue)
		}
		return issues, nil
	}
	return nil, nil
}
