package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestPathOption(t *testing.T) {
	entry := &rules.FstabEntry{LineNum: 1, RawLine: "test"}

	tests := []struct {
		name     string
		opt      options.PathOption
		val      string
		hasVal   bool
		expected int
	}{
		{
			name:     "missing value",
			opt:      options.PathOption{RequireAbsolute: true},
			val:      "",
			hasVal:   false,
			expected: 1,
		},
		{
			name:     "empty value",
			opt:      options.PathOption{RequireAbsolute: true},
			val:      "",
			hasVal:   true,
			expected: 1,
		},
		{
			name:     "absolute path required and provided",
			opt:      options.PathOption{RequireAbsolute: true},
			val:      "/absolute/path",
			hasVal:   true,
			expected: 0,
		},
		{
			name:     "absolute path required but relative provided",
			opt:      options.PathOption{RequireAbsolute: true},
			val:      "relative/path",
			hasVal:   true,
			expected: 1,
		},
		{
			name:     "absolute path not required and relative provided",
			opt:      options.PathOption{RequireAbsolute: false},
			val:      "relative/path",
			hasVal:   true,
			expected: 0,
		},
		{
			name:     "absolute path not required and absolute provided",
			opt:      options.PathOption{RequireAbsolute: false},
			val:      "/absolute/path",
			hasVal:   true,
			expected: 0,
		},
		{
			name:     "escaped space in absolute path",
			opt:      options.PathOption{RequireAbsolute: true},
			val:      "/absolute/path\\040with\\040space",
			hasVal:   true,
			expected: 0, // unescapes to "/absolute/path with space" which is absolute
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := tt.opt.ValidateOption(context.Background(), nil, entry, "ext4", "target-prefix", tt.val, tt.hasVal, nil)
			if len(issues) != tt.expected {
				t.Errorf("expected %d issues, got %d: %v", tt.expected, len(issues), issues)
			}
		})
	}
}
