package rcs

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rcslib "github.com/arran4/golang-rcs"
)

// Verify an actual earlier revision through the existing RCS checkout library.
func TestHistoricalRevisionMustBeRestorable(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(t.TempDir(), "fstab")
	for _, content := range []string{"original\n", "edited\n"} {
		if err := Update(source, []byte(content), Config{Dir: dir}); err != nil {
			t.Fatal(err)
		}
	}
	history, err := HistoryPath(source, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(history)
	if err != nil {
		t.Fatal(err)
	}
	f, err := rcslib.ParseFile(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	old, err := f.Checkout("", rcslib.WithRevision("1.1"))
	if err != nil {
		t.Fatalf("previous revision cannot be restored: %v", err)
	}
	if old.Content != "original\n" {
		t.Fatalf("restored %q", old.Content)
	}
}

func TestSameBasenameHistoriesStaySeparate(t *testing.T) {
	dir := t.TempDir()
	for _, source := range []string{"/fixtures/one/fstab", "/fixtures/two/fstab"} {
		if err := Update(source, []byte(source), Config{Dir: dir}); err != nil {
			t.Fatal(err)
		}
	}
	count := 0
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ",v") {
			count++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("different source paths share history: got %d archive(s), want 2", count)
	}
}
