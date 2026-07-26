package core

import (
	"errors"
	"strings"
	"testing"
	"github.com/arran4/vifstab/internal/rules"
)

var errMockRead = errors.New("mock read error")

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errMockRead
}

func TestParseFstab(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []*rules.FstabEntry
		wantErr bool
	}{
		{
			name:  "empty input",
			input: "",
			want:  []*rules.FstabEntry{},
		},
		{
			name:  "single valid entry",
			input: "/dev/sda1 / ext4 defaults 0 1",
			want: []*rules.FstabEntry{
				{
					RawLine: "/dev/sda1 / ext4 defaults 0 1",
					LineNum: 1,
					Fields:  []string{"/dev/sda1", "/", "ext4", "defaults", "0", "1"},
				},
			},
		},
		{
			name:  "multiple entries",
			input: "/dev/sda1 / ext4 defaults 0 1\n/dev/sda2 none swap sw 0 0",
			want: []*rules.FstabEntry{
				{
					RawLine: "/dev/sda1 / ext4 defaults 0 1",
					LineNum: 1,
					Fields:  []string{"/dev/sda1", "/", "ext4", "defaults", "0", "1"},
				},
				{
					RawLine: "/dev/sda2 none swap sw 0 0",
					LineNum: 2,
					Fields:  []string{"/dev/sda2", "none", "swap", "sw", "0", "0"},
				},
			},
		},
		{
			name:  "with comments and blank lines",
			input: "# this is a comment\n\n/dev/sda1 / ext4 defaults 0 1\n# another comment",
			want: []*rules.FstabEntry{
				{
					RawLine: "# this is a comment",
					LineNum: 1,
					Fields:  []string{"#", "this", "is", "a", "comment"},
				},
				{
					RawLine: "",
					LineNum: 2,
					Fields:  []string{},
				},
				{
					RawLine: "/dev/sda1 / ext4 defaults 0 1",
					LineNum: 3,
					Fields:  []string{"/dev/sda1", "/", "ext4", "defaults", "0", "1"},
				},
				{
					RawLine: "# another comment",
					LineNum: 4,
					Fields:  []string{"#", "another", "comment"},
				},
			},
		},
		{
			name:  "with tabs and multiple spaces",
			input: "/dev/sda1\t/\t  ext4   defaults\t0\t1",
			want: []*rules.FstabEntry{
				{
					RawLine: "/dev/sda1\t/\t  ext4   defaults\t0\t1",
					LineNum: 1,
					Fields:  []string{"/dev/sda1", "/", "ext4", "defaults", "0", "1"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFstab(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFstab() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) == 0 && len(tt.want) == 0 {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("ParseFstab() got %d entries, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].RawLine != tt.want[i].RawLine {
					t.Errorf("ParseFstab()[%d].RawLine = %v, want %v", i, got[i].RawLine, tt.want[i].RawLine)
				}
				if got[i].LineNum != tt.want[i].LineNum {
					t.Errorf("ParseFstab()[%d].LineNum = %v, want %v", i, got[i].LineNum, tt.want[i].LineNum)
				}

				if len(got[i].Fields) != len(tt.want[i].Fields) {
					t.Errorf("ParseFstab()[%d].Fields length = %d, want %d", i, len(got[i].Fields), len(tt.want[i].Fields))
				} else {
					for j := range got[i].Fields {
						if got[i].Fields[j] != tt.want[i].Fields[j] {
							t.Errorf("ParseFstab()[%d].Fields[%d] = %v, want %v", i, j, got[i].Fields[j], tt.want[i].Fields[j])
						}
					}
				}
			}
		})
	}

	t.Run("reader error", func(t *testing.T) {
		_, err := ParseFstab(&errorReader{})
		if err == nil {
			t.Errorf("ParseFstab() expected error, got nil")
		} else if !errors.Is(err, errMockRead) {
			t.Errorf("ParseFstab() expected %v, got %v", errMockRead, err)
		}
	})
}
