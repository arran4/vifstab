package editor

import (
	"context"
	"fmt"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin" // Register rules
)

func lintEntriesBaseline(entries []*rules.FstabEntry, api rules.API) ([]rules.ValidationIssue, error) {
	allRules := rules.GetAllRules()
	ctx := rules.WithCache(context.Background(), rules.NewCache())

	var allIssues []rules.ValidationIssue
	for _, entry := range entries {
		for _, r := range allRules {
			issues, err := r.Validate(ctx, api, entry)
			if err != nil {
				return nil, fmt.Errorf("rule %s failed: %w", r.Name(), err)
			}
			allIssues = append(allIssues, issues...)
		}
	}

	return allIssues, nil
}

type mockAPI struct{}

func (m *mockAPI) GetFstabEntries() []*rules.FstabEntry                        { return nil }
func (m *mockAPI) HasOption(entry *rules.FstabEntry, opt string) bool          { return false }
func (m *mockAPI) IsOptionValue(entry *rules.FstabEntry, opt, val string) bool { return false }
func (m *mockAPI) CheckFileExists(path string) rules.FileExistsResult          { return rules.FileExists }
func (m *mockAPI) CheckBlockDeviceExists(dev string) bool                      { return true }
func (m *mockAPI) GetHostKernelVersion() string                                { return "5.15.0" }
func (m *mockAPI) CheckUserExists(uid int) bool                                { return true }
func (m *mockAPI) CheckGroupExists(gid int) bool                               { return true }

func validateFixture(b *testing.B, entries []*rules.FstabEntry, expectedIssues int) {
	b.Helper()
	api := &mockAPI{}
	issues, err := lintEntries(entries, api)
	if err != nil {
		b.Fatalf("setup validation failed: %v", err)
	}
	if got, want := len(issues), expectedIssues; got != want {
		b.Fatalf("benchmark fixture produced %d issues, want %d", got, want)
	}
}

func BenchmarkLintEntries_ZeroIssues(b *testing.B) {
	var entries []*rules.FstabEntry
	for i := 0; i < 10000; i++ {
		entries = append(entries, &rules.FstabEntry{
			Fields: []string{"/dev/sda1", "/", "ext4", "defaults", "0", "1"},
		})
	}

	validateFixture(b, entries, 0)

	api := &mockAPI{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lintEntries(entries, api)
	}
}

func BenchmarkLintEntries_OneTotalIssue(b *testing.B) {
	var entries []*rules.FstabEntry
	for i := 0; i < 10000; i++ {
		entry := &rules.FstabEntry{
			Fields: []string{"/dev/sda1", "/", "ext4", "defaults", "0", "1"},
		}
		if i == 5000 {
			entry.Fields[3] = "defaults,badopt"
		}
		entries = append(entries, entry)
	}

	validateFixture(b, entries, 1)

	api := &mockAPI{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lintEntries(entries, api)
	}
}

func BenchmarkLintEntries_OneIssuePerEntry(b *testing.B) {
	var entries []*rules.FstabEntry
	for i := 0; i < 10000; i++ {
		entries = append(entries, &rules.FstabEntry{
			Fields: []string{"/dev/sda1", "/", "ext4", "defaults,badopt", "0", "1"},
		})
	}

	validateFixture(b, entries, 10000)

	api := &mockAPI{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lintEntries(entries, api)
	}
}

func BenchmarkLintEntries_MultipleIssuesPerEntry(b *testing.B) {
	var entries []*rules.FstabEntry
	for i := 0; i < 10000; i++ {
		entries = append(entries, &rules.FstabEntry{
			Fields: []string{"none", "/invalid", "ext4", "badopt1,badopt2,badopt3", "9", "9"},
		})
	}

	api := &mockAPI{}
	issues, err := lintEntries(entries, api)
	if err != nil {
		b.Fatalf("setup validation failed: %v", err)
	}
	expected := len(issues)

	if expected <= len(entries) {
		b.Fatalf("expected issue-heavy fixture to produce more issues than entries; got %d issues for %d entries", expected, len(entries))
	}

	validateFixture(b, entries, expected)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lintEntries(entries, api)
	}
}

func BenchmarkLintEntries_ZeroIssues_Baseline(b *testing.B) {
	var entries []*rules.FstabEntry
	for i := 0; i < 10000; i++ {
		entries = append(entries, &rules.FstabEntry{
			Fields: []string{"/dev/sda1", "/", "ext4", "defaults", "0", "1"},
		})
	}
	api := &mockAPI{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lintEntriesBaseline(entries, api)
	}
}

func BenchmarkLintEntries_OneTotalIssue_Baseline(b *testing.B) {
	var entries []*rules.FstabEntry
	for i := 0; i < 10000; i++ {
		entry := &rules.FstabEntry{
			Fields: []string{"/dev/sda1", "/", "ext4", "defaults", "0", "1"},
		}
		if i == 5000 {
			entry.Fields[3] = "defaults,badopt"
		}
		entries = append(entries, entry)
	}
	api := &mockAPI{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lintEntriesBaseline(entries, api)
	}
}

func BenchmarkLintEntries_OneIssuePerEntry_Baseline(b *testing.B) {
	var entries []*rules.FstabEntry
	for i := 0; i < 10000; i++ {
		entries = append(entries, &rules.FstabEntry{
			Fields: []string{"/dev/sda1", "/", "ext4", "defaults,badopt", "0", "1"},
		})
	}
	api := &mockAPI{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lintEntriesBaseline(entries, api)
	}
}

func BenchmarkLintEntries_MultipleIssuesPerEntry_Baseline(b *testing.B) {
	var entries []*rules.FstabEntry
	for i := 0; i < 10000; i++ {
		entries = append(entries, &rules.FstabEntry{
			Fields: []string{"none", "/invalid", "ext4", "badopt1,badopt2,badopt3", "9", "9"},
		})
	}
	api := &mockAPI{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lintEntriesBaseline(entries, api)
	}
}
