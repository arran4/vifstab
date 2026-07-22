package editor

import (
	"context"
	"fmt"
	"os"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin/all"
	_ "github.com/arran4/vifstab/internal/rules/builtin/all" // Register rules
	"github.com/arran4/vifstab/internal/rules/core"
)

// VerifyAndLint reads the given file and lints it against all registered rules.
// It returns a slice of all suggestions (warnings, errors, etc.) and an error if a technical failure occurred.
func VerifyAndLint(cfg config.Config) ([]rules.ValidationIssue, error) {
	f, err := os.Open(cfg.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file for verification: %w", err)
	}
	defer f.Close()

	entries, err := core.ParseFstab(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	// Run rules
	allRules := rules.GetAllRules()
	ctx := rules.WithCache(context.Background(), rules.NewCache())

	var allIssues []rules.ValidationIssue
	for _, entry := range entries {
		for _, r := range allRules {
			issues, err := r.Validate(ctx, entry)
			if err != nil {
				return nil, fmt.Errorf("rule %s failed: %w", r.Name(), err)
			}
			allIssues = append(allIssues, issues...)
		}
	}

	return allIssues, nil
}
