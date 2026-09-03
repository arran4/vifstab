package format

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
)

//go:embed testdata/*.txtar
var testdataFS embed.FS

func TestFormat_Txtar(t *testing.T) {
	entries, err := fs.ReadDir(testdataFS, "testdata")
	if err != nil {
		t.Fatalf("failed to read testdata dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txtar") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			b, err := fs.ReadFile(testdataFS, "testdata/"+entry.Name())
			if err != nil {
				t.Fatalf("failed to read test file: %v", err)
			}

			archive := txtar.Parse(b)
			var input string
			var expected string

			for _, file := range archive.Files {
				switch file.Name {
				case "input/etc/fstab":
					input = string(file.Data)
				case "expected/format.txt":
					expected = string(file.Data)
				}
			}

			if input == "" {
				t.Fatalf("missing input/etc/fstab in test archive")
			}
			if expected == "" {
				t.Fatalf("missing expected/format.txt in test archive")
			}

			r := strings.NewReader(input)
			var w bytes.Buffer

			if err := Format(r, &w); err != nil {
				t.Fatalf("Format failed: %v", err)
			}

			gotStr := strings.TrimSpace(w.String())
			expectedStr := strings.TrimSpace(expected)

			if gotStr != expectedStr {
				t.Errorf("Format result incorrect.\nGot:\n%s\nExpected:\n%s", gotStr, expectedStr)
			}
		})
	}
}

type errorWriter struct{}

func (w errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("injected write error")
}

func TestFormat_WriteError(t *testing.T) {
	input := "proc /proc proc defaults 0 0\n"
	err := Format(strings.NewReader(input), errorWriter{})
	if err == nil || !strings.Contains(err.Error(), "injected write error") {
		t.Fatalf("expected injected write error, got %v", err)
	}
}
