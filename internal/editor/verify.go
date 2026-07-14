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
// It returns a count of validation issues found and an error if a technical failure occurred.
func VerifyAndLint(cfg config.Config) (int, error) {
	f, err := os.Open(cfg.FilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read file for verification: %w", err)
	}
	defer f.Close()

	entries, err := core.ParseFstab(f)
	if err != nil {
		return 0, fmt.Errorf("failed to parse: %w", err)
	}

	// Run rules
	allRules := rules.GetAllRules()
	ctx := rules.WithCache(context.Background(), rules.NewCache())

	var allSuggestions []rules.Suggestion
	for _, entry := range entries {
		for _, r := range allRules {
			suggestions, err := r.Validate(ctx, entry)
			if err != nil {
				return 0, fmt.Errorf("rule %s failed: %w", r.Name(), err)
			}
			allSuggestions = append(allSuggestions, suggestions...)
		}
	}

	var issueCount int
	for _, s := range allSuggestions {
		if s.Level == rules.SuggestionLevelError || s.Level == rules.SuggestionLevelWarning {
			issueCount++
		}
	}

	if issueCount == 0 {
		fmt.Println("Validation successful.")
		return 0, nil
	}

	fmt.Println("Validation issues found:")
	for _, s := range allSuggestions {
		fmt.Printf(" - [%s] %s (Line: %s)\n", s.Level, s.Message, s.Line)
	}

	return issueCount, nil
}
