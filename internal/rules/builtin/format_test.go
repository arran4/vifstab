package builtin

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
)

func TestFormatRule_Validate(t *testing.T) {
	r := &FormatRule{}
	ctx := context.Background()

	tests := []struct {
		name     string
		entry    *rules.FstabEntry
		wantErrs int
		wantErr  bool
	}{
		{
			name: "valid entry with 6 fields",
			entry: &rules.FstabEntry{
				LineNum: 1,
				RawLine: "/dev/sda1 / ext4 defaults 0 0",
				Fields:  []string{"/dev/sda1", "/", "ext4", "defaults", "0", "0"},
			},
			wantErrs: 0,
			wantErr:  false,
		},
		{
			name: "empty entry",
			entry: &rules.FstabEntry{
				LineNum: 2,
				RawLine: "",
				Fields:  []string{},
			},
			wantErrs: 0,
			wantErr:  false,
		},
		{
			name: "comment entry",
			entry: &rules.FstabEntry{
				LineNum: 3,
				RawLine: "# this is a comment",
				Fields:  []string{"#", "this", "is", "a", "comment"},
			},
			wantErrs: 0,
			wantErr:  false,
		},
		{
			name: "less than 6 fields",
			entry: &rules.FstabEntry{
				LineNum: 4,
				RawLine: "/dev/sda1 / ext4 defaults 0",
				Fields:  []string{"/dev/sda1", "/", "ext4", "defaults", "0"},
			},
			wantErrs: 1,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, err := r.Validate(ctx, nil, tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatRule.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(issues) != tt.wantErrs {
				t.Errorf("FormatRule.Validate() got %d issues, want %d", len(issues), tt.wantErrs)
			}
			if tt.wantErrs > 0 && len(issues) > 0 {
				if issues[0].Level != rules.IssueLevelError {
					t.Errorf("FormatRule.Validate() issue level = %v, want %v", issues[0].Level, rules.IssueLevelError)
				}
				expectedMsg := "Line 4 has less than 6 fields"
				if tt.name == "less than 6 fields" && issues[0].Message != expectedMsg {
					t.Errorf("FormatRule.Validate() issue message = %q, want %q", issues[0].Message, expectedMsg)
				}
			}
		})
	}
}
