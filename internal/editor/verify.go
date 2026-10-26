package editor

import (
	"context"
	"fmt"
	"io"
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

func (m *memoizedAPI) GetSupportedFilesystems() (map[string]bool, error) {
	return rules.SupportedFilesystems(m.API)
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
	return VerifyAndLintReader(f, api)
}

// VerifyAndLintReader lints fstab entries directly from an io.Reader.
func VerifyAndLintReader(r io.Reader, api rules.API) ([]rules.ValidationIssue, error) {
	entries, err := core.ParseFstab(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	// Run rules
	return lintEntries(entries, api)
}

// lintEntries validates a list of fstab entries against all registered rules.
func lintEntries(entries []*rules.FstabEntry, api rules.API) ([]rules.ValidationIssue, error) {
	return lintEntriesWithContext(context.Background(), entries, api)
}

func lintEntriesWithContext(ctx context.Context, entries []*rules.FstabEntry, api rules.API) ([]rules.ValidationIssue, error) {
	allRules := rules.GetAllRules()

	if rules.CacheFromContext(ctx) == nil {
		cache := rules.NewCache()
		if provider, ok := api.(interface{ ValidationCache() *rules.Cache }); ok {
			if captured := provider.ValidationCache(); captured != nil {
				cache = captured
			}
		}
		ctx = rules.WithCache(ctx, cache)
	}

	memoized := &memoizedAPI{API: api}

	var allIssues []rules.ValidationIssue
	for _, entry := range entries {
		for _, r := range allRules {
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
