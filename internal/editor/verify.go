package editor

import (
	"context"
	"fmt"
	"os"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin" // Register rules
	"github.com/arran4/vifstab/internal/rules/core"
)

// VerifyAndLint reads the given file and lints it against all registered rules.
// It returns a slice of all suggestions (warnings, errors, etc.) and an error if a technical failure occurred.
func VerifyAndLint(cfg config.Config, api rules.API) ([]rules.ValidationIssue, error) {
	f, err := os.Open(cfg.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file for verification: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	entries, err := core.ParseFstab(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	// Run rules
	return lintEntries(entries, api)
}

// lintEntries validates a list of fstab entries against all registered rules.
func lintEntries(entries []*rules.FstabEntry, api rules.API) ([]rules.ValidationIssue, error) {
	allRules := rules.GetAllRules()
	ctx := rules.WithCache(context.Background(), rules.NewCache())

	var allIssues []rules.ValidationIssue
	for _, entry := range entries {
		for _, r := range allRules {
			issues, err := r.Validate(ctx, api, entry)
			if err != nil {
				return nil, fmt.Errorf("rule %s failed: %w", r.Name(), err)
			}
			if len(issues) > 0 && allIssues == nil {
				allIssues = make([]rules.ValidationIssue, 0, len(entries))
			}
			allIssues = append(allIssues, issues...)
		}
	}

	return allIssues, nil
}
