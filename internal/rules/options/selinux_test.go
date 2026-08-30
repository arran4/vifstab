package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestSELinuxContextOption(t *testing.T) {
	ctx := context.Background()
	validator := options.SELinuxContextOption{}
	entry := &rules.FstabEntry{LineNum: 1, RawLine: "mock_line"}
	fsName := "mockfs"

	tests := []struct {
		name     string
		key      string
		val      string
		hasVal   bool
		expected int
	}{
		{
			name:     "valid context (3 parts)",
			key:      "context",
			val:      "system_u:object_r:tmpfs_t",
			hasVal:   true,
			expected: 0,
		},
		{
			name:     "valid context (4 parts)",
			key:      "context",
			val:      "system_u:object_r:tmpfs_t:s0",
			hasVal:   true,
			expected: 0,
		},
		{
			name:     "missing value",
			key:      "context",
			val:      "",
			hasVal:   false,
			expected: 1,
		},
		{
			name:     "empty value",
			key:      "context",
			val:      "",
			hasVal:   true,
			expected: 1,
		},
		{
			name:     "invalid context (2 parts)",
			key:      "context",
			val:      "system_u:object_r",
			hasVal:   true,
			expected: 1,
		},
		{
			name:     "invalid context (5 parts)",
			key:      "context",
			val:      "system_u:object_r:tmpfs_t:s0:extra",
			hasVal:   true,
			expected: 1,
		},
		{
			name:     "invalid context (empty field)",
			key:      "context",
			val:      "system_u::tmpfs_t",
			hasVal:   true,
			expected: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issues := validator.ValidateOption(ctx, nil, entry, fsName, tc.key, tc.val, tc.hasVal, nil)
			if len(issues) != tc.expected {
				t.Errorf("expected %d issues, got %d", tc.expected, len(issues))
			}
		})
	}
}
