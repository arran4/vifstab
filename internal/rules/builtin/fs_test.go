package builtin

import (
	"context"
	"embed"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/core"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/fs.txtar
var testdataFS embed.FS

func TestFSRule(t *testing.T) {
	data, err := testdataFS.ReadFile("testdata/fs.txtar")
	if err != nil {
		t.Fatalf("failed to read txtar: %v", err)
	}

	ar := txtar.Parse(data)

	var procFS []byte
	var validFstab []byte
	var invalidFstab []byte

	for _, f := range ar.Files {
		switch f.Name {
		case "proc_filesystems":
			procFS = f.Data
		case "valid_fstab":
			validFstab = f.Data
		case "invalid_fstab":
			invalidFstab = f.Data
		}
	}

	fsMap := make(map[string]bool)
	lines := strings.Split(string(procFS), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			fsMap[fields[len(fields)-1]] = true
		}
	}

	cache := rules.NewCache()
	cache.Set(ProcFilesystemsKey, fsMap)
	ctx := rules.WithCache(context.Background(), cache)

	rule := &FSRule{}

	// Test Valid
	validEntries, err := core.ParseFstab(strings.NewReader(string(validFstab)))
	if err != nil {
		t.Fatalf("failed to parse valid fstab: %v", err)
	}

	for _, entry := range validEntries {
		suggs, err := rule.Validate(ctx, entry)
		if err != nil {
			t.Fatalf("unexpected error during valid validate: %v", err)
		}
		if len(suggs) > 0 {
			t.Errorf("expected valid entry to have no suggestions, got %v for line: %s", suggs, entry.RawLine)
		}
	}

	// Test Invalid
	invalidEntries, err := core.ParseFstab(strings.NewReader(string(invalidFstab)))
	if err != nil {
		t.Fatalf("failed to parse invalid fstab: %v", err)
	}

	expectedMsgs := []string{
		"Unknown filesystem type 'zfs'",
		"Filesystem type for swap must be 'swap'",
	}
	foundMsgs := make(map[string]bool)
	for _, entry := range invalidEntries {
		suggs, err := rule.Validate(ctx, entry)
		if err != nil {
			t.Fatalf("unexpected error during invalid validate: %v", err)
		}
		for _, sugg := range suggs {
			for _, exp := range expectedMsgs {
				if strings.Contains(sugg.Message, exp) {
					foundMsgs[exp] = true
				}
			}
		}
	}

	for _, exp := range expectedMsgs {
		if !foundMsgs[exp] {
			t.Errorf("expected to find error message containing %q, but did not", exp)
		}
	}
}
