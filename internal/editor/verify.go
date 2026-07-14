package editor

import (
	"context"
	"fmt"
	"os"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/outcome"
	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin" // Register rules
	"github.com/arran4/vifstab/internal/rules/core"
)

// VerifyAndLint reads the given file and lints it against all registered rules.
// It returns an error if any validation issues are found, without prompting the user or modifying the file.
func VerifyAndLint(cfg config.Config) error {
	f, err := os.Open(cfg.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file for verification: %w", err)
	}
	defer f.Close()

	entries, err := core.ParseFstab(f)
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	// Run rules
	allRules := rules.GetAllRules()
	ctx := rules.WithCache(context.Background(), rules.NewCache())

	var allSuggestions []rules.Suggestion
	for _, entry := range entries {
		for _, r := range allRules {
			suggestions, err := r.Validate(ctx, entry)
			if err != nil {
				return fmt.Errorf("rule %s failed: %w", r.Name(), err)
			}
			allSuggestions = append(allSuggestions, suggestions...)
		}
	}

	if len(allSuggestions) == 0 {
		fmt.Println("Validation successful.")
		return nil
	}

	fmt.Println("Validation issues found:")
	for _, s := range allSuggestions {
		fmt.Printf(" - [%s] %s (Line: %s)\n", s.Level, s.Message, s.Line)
	}

	return outcome.New(1, fmt.Errorf("validation failed with %d issues", len(allSuggestions)))
}
