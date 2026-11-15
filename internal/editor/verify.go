package editor

import (
	"context"
	"fmt"
	"os"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin" // Register rules
	"github.com/arran4/vifstab/internal/rules/core"
	"sync"
)

type memoizedAPI struct {
	rules.API
	once sync.Once
	inv  rules.BlockDeviceInventory
	err  error
}

func (m *memoizedAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	m.once.Do(func() {
		m.inv, m.err = m.API.GetBlockDevices()
	})
	return m.inv, m.err
}

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
	ctx := rules.WithTargetFstab(context.Background(), cfg.FilePath)
	return lintEntriesWithContext(ctx, entries, api)
}

// lintEntries validates a list of fstab entries against all registered rules.
func lintEntries(entries []*rules.FstabEntry, api rules.API) ([]rules.ValidationIssue, error) {
	return lintEntriesWithContext(context.Background(), entries, api)
}

func lintEntriesWithContext(ctx context.Context, entries []*rules.FstabEntry, api rules.API) ([]rules.ValidationIssue, error) {
	allRules := rules.GetAllRules()

	if rules.CacheFromContext(ctx) == nil {
		ctx = rules.WithCache(ctx, rules.NewCache())
	}

	memoized := &memoizedAPI{API: api}

	var allIssues []rules.ValidationIssue
	for _, r := range allRules {
		if fr, ok := r.(rules.FileRule); ok {
			issues, err := fr.ValidateFile(ctx, memoized)
			if err != nil {
				return nil, fmt.Errorf("rule %s failed: %w", r.Name(), err)
			}
			allIssues = append(allIssues, issues...)
		}
	}
	for _, entry := range entries {
		for _, r := range allRules {
			if _, ok := r.(rules.FileRule); ok {
				continue
			}
			issues, err := r.Validate(ctx, memoized, entry)
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
