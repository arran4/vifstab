package editor

import (
	"testing"
	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin" // Register rules
)

type mockAPI struct{}
func (m *mockAPI) GetFstabEntries() []*rules.FstabEntry { return nil }
func (m *mockAPI) HasOption(entry *rules.FstabEntry, opt string) bool { return false }
func (m *mockAPI) IsOptionValue(entry *rules.FstabEntry, opt, val string) bool { return false }
func (m *mockAPI) CheckFileExists(path string) rules.FileExistsResult { return rules.FileExists }
func (m *mockAPI) CheckBlockDeviceExists(dev string) bool { return true }
func (m *mockAPI) GetHostKernelVersion() string { return "5.15.0" }
func (m *mockAPI) CheckUserExists(uid int) bool { return true }
func (m *mockAPI) CheckGroupExists(gid int) bool { return true }

func BenchmarkLintEntries_ZeroIssues(b *testing.B) {
	var entries []*rules.FstabEntry
	for i := 0; i < 100; i++ {
		entries = append(entries, &rules.FstabEntry{
			Fields: []string{"/dev/sda1", "/", "ext4", "defaults", "0", "1"},
		})
	}
	api := &mockAPI{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lintEntries(entries, api)
	}
}

func BenchmarkLintEntries_OneIssuePerEntry(b *testing.B) {
	var entries []*rules.FstabEntry
	for i := 0; i < 100; i++ {
		entries = append(entries, &rules.FstabEntry{
			Fields: []string{"/dev/sda1", "/", "ext4", "defaults,badopt", "0", "1"},
		})
	}
	api := &mockAPI{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lintEntries(entries, api)
	}
}

func BenchmarkLintEntries_MultipleIssuesPerEntry(b *testing.B) {
	var entries []*rules.FstabEntry
	for i := 0; i < 100; i++ {
		entries = append(entries, &rules.FstabEntry{
			Fields: []string{"none", "/invalid", "unknownfs", "badopt1,badopt2,badopt3", "9", "9"},
		})
	}
	api := &mockAPI{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lintEntries(entries, api)
	}
}
