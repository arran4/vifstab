package builtin

import (
	"context"
	"reflect"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
)

func TestFormatRule_Validate(t *testing.T) {
	r := &FormatRule{}
	ctx := context.Background()

	tests := []struct {
		name       string
		entry      *rules.FstabEntry
		wantIssues []rules.ValidationIssue
	}{
		{
			name: "valid entry with 6 fields",
			entry: &rules.FstabEntry{
				LineNum: 1,
				RawLine: "/dev/sda1 / ext4 defaults 0 0",
				Fields:  []string{"/dev/sda1", "/", "ext4", "defaults", "0", "0"},
			},
			wantIssues: nil,
		},
		{
			name: "empty entry",
			entry: &rules.FstabEntry{
				LineNum: 2,
				RawLine: "",
				Fields:  []string{},
			},
			wantIssues: nil,
		},
		{
			name: "comment entry with leading space",
			entry: &rules.FstabEntry{
				LineNum: 3,
				RawLine: "   # this is a comment",
				Fields:  []string{"#", "this", "is", "a", "comment"},
			},
			wantIssues: nil,
		},
		{
			name: "less than 6 fields",
			entry: &rules.FstabEntry{
				LineNum: 4,
				RawLine: "/dev/sda1 / ext4 defaults 0",
				Fields:  []string{"/dev/sda1", "/", "ext4", "defaults", "0"},
			},
			wantIssues: []rules.ValidationIssue{
				{
					Message: "Line 4 has less than 6 fields",
					Line:    "/dev/sda1 / ext4 defaults 0",
					Level:   rules.IssueLevelError,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, err := r.Validate(ctx, nil, tt.entry)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(issues) == 0 && len(tt.wantIssues) == 0 {
				return
			}
			if !reflect.DeepEqual(issues, tt.wantIssues) {
				t.Errorf("FormatRule.Validate() issues = %v, want %v", issues, tt.wantIssues)
			}
		})
	}
}
