package format

import (
	"bytes"
	_ "embed"
	"strings"
	"testing"
)

//go:embed testdata/input.txt
var input string

//go:embed testdata/expected.txt
var expected string

func TestFormat(t *testing.T) {
	r := strings.NewReader(input)
	var w bytes.Buffer

	if err := Format(r, &w); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if got := w.String(); got != expected {
		t.Errorf("Format result incorrect.\nGot:\n%s\nExpected:\n%s", got, expected)
	}
}
