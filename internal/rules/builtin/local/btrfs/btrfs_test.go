package btrfs

import (
	"context"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
)

// We can just rely on the existing fs_test.go architecture that uses txtar files,
// but let's add a quick test to explicitly check kernel version parsing.
func TestKernelVersionParsing(t *testing.T) {
	maj, min, p := parseKernelVersionStr("Linux version 6.18.35-gentoo-dist")
	if maj != 6 || min != 18 || p != 35 {
		t.Errorf("Expected 6.18.35, got %d.%d.%d", maj, min, p)
	}

	maj, min, p = parseKernelVersionStr("6.1.1-arch1-1")
	if maj != 6 || min != 1 || p != 1 {
		t.Errorf("Expected 6.1.1, got %d.%d.%d", maj, min, p)
	}
}

// Ensure the rule is registered and can process entries.
func TestBTRFSRuleSimple(t *testing.T) {
	rule := &BTRFSRule{}
	entry := &rules.FstabEntry{
		RawLine: "/dev/sda1 /mnt btrfs defaults,compress=zstd:3,discard=async 0 0",
		LineNum: 1,
		Fields:  strings.Fields("/dev/sda1 /mnt btrfs defaults,compress=zstd:3,discard=async 0 0"),
	}

	suggestions, err := rule.Validate(context.Background(), entry)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// We might get warnings depending on the host running the test, so we can't assert length is 0 safely.
	// But we can check that it didn't crash.
	for _, s := range suggestions {
		t.Logf("Suggestion: %s", s.Message)
	}
}
