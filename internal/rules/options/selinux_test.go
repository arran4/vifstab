package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestSELinuxContextOption(t *testing.T) {
	ctx := context.Background()
	entry := &rules.FstabEntry{LineNum: 1, RawLine: "test"}
	fsName := "ext4"
	opt := options.SELinuxContextOption{}

	tests := []struct {
		name    string
		key     string
		val     string
		hasVal  bool
		wantErr bool
	}{
		{"Valid context 3 parts", "context", "system_u:object_r:tmpfs_t", true, false},
		{"Valid context 4 parts", "context", "system_u:object_r:tmpfs_t:s0", true, false},
		{"Missing value", "context", "", false, true},
		{"Empty string", "context", "", true, true},
		{"Too few parts", "context", "system_u:object_r", true, true},
		{"Valid with MLS categories", "context", "system_u:object_r:tmpfs_t:s0:c0,c1", true, false},
		{"Empty parts", "context", "system_u::tmpfs_t:s0", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := opt.ValidateOption(ctx, nil, entry, fsName, tt.key, tt.val, tt.hasVal, nil)
			if tt.wantErr && len(issues) == 0 {
				t.Errorf("expected error, got none")
			}
			if !tt.wantErr && len(issues) > 0 {
				t.Errorf("expected no error, got %v", issues)
			}
		})
	}
}
