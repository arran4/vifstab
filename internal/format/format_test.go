package format

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	input := `/dev/sda1 / ext4 defaults 0 1
/dev/mapper/root / ext4 rw,noatime 0 1
# This is a very very long comment that should not stretch the columns
tmpfs /tmp tmpfs defaults,noatime,mode=1777 0 0
`
	expected := `/dev/sda1         /  ext4  defaults    0  1
/dev/mapper/root  /  ext4  rw,noatime  0  1
# This is a very very long comment that should not stretch the columns
tmpfs  /tmp  tmpfs  defaults,noatime,mode=1777  0  0
`

	r := strings.NewReader(input)
	var w bytes.Buffer

	if err := Format(r, &w); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if got := w.String(); got != expected {
		t.Errorf("Format result incorrect.\nGot:\n%s\nExpected:\n%s", got, expected)
	}
}
