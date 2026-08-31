package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestStringOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 5, RawLine: "LABEL=root / ext4 rw 0 1"}
	opt := options.StringOption{}

	t.Run("valid string option with value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "label", "root", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected no issues, got %d", len(issues))
		}
	})

	t.Run("missing value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "label", "", false, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Missing value for ext4 mount option 'label' on line 5"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected msg %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}

func TestSELinuxContextOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 5, RawLine: "LABEL=root / ext4 context=system_u:object_r:tmpfs_t:s0 0 1"}
	opt := options.SELinuxContextOption{}

	tests := []struct {
		name     string
		val      string
		hasVal   bool
		wantErrs int
		wantErrMsg string
	}{
		{
			name:   "valid unquoted",
			val:    "system_u:object_r:tmpfs_t:s0",
			hasVal: true,
		},
		{
			name:   "valid unquoted no mls",
			val:    "system_u:object_r:tmpfs_t",
			hasVal: true,
		},
		{
			name:   "valid quoted",
			val:    `"system_u:object_r:tmpfs_t:s0"`,
			hasVal: true,
		},
		{
			name:   "valid with complex mls",
			val:    "system_u:object_r:svirt_image_t:s0:c123,c456",
			hasVal: true,
		},
		{
			name:   "valid with complex mls range",
			val:    "system_u:object_r:svirt_image_t:s0-s0:c0.c1023",
			hasVal: true,
		},
		{
			name:       "missing value",
			val:        "",
			hasVal:     false,
			wantErrs:   1,
			wantErrMsg: "Missing value for ext4 mount option 'context' on line 5",
		},
		{
			name:       "invalid context",
			val:        "invalid_context",
			hasVal:     true,
			wantErrs:   1,
			wantErrMsg: "Invalid SELinux context 'invalid_context' for ext4 mount option 'context' on line 5",
		},
		{
			name:       "invalid context parts",
			val:        "user:role",
			hasVal:     true,
			wantErrs:   1,
			wantErrMsg: "Invalid SELinux context 'user:role' for ext4 mount option 'context' on line 5",
		},
		{
			name:       "unbalanced quotes",
			val:        `"system_u:object_r:tmpfs_t:s0`,
			hasVal:     true,
			wantErrs:   1,
			wantErrMsg: "Malformed or unbalanced quotes in ext4 mount option 'context' on line 5",
		},
		{
			name:       "context=@target rejected",
			val:        `@target`,
			hasVal:     true,
			wantErrs:   1,
			wantErrMsg: "Special value '@target' is only allowed for 'rootcontext', invalid in ext4 mount option 'context' on line 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := opt.ValidateOption(ctx, api, entry, "ext4", "context", tt.val, tt.hasVal, nil)
			if len(issues) != tt.wantErrs {
				t.Fatalf("expected %d issues, got %d", tt.wantErrs, len(issues))
			}
			if tt.wantErrs > 0 {
				if issues[0].Message != tt.wantErrMsg {
					t.Errorf("expected msg %q, got %q", tt.wantErrMsg, issues[0].Message)
				}
			}
		})
	}

	t.Run("rootcontext=@target allowed", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "rootcontext", "@target", true, nil)
		if len(issues) != 0 {
			t.Fatalf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("rootcontext=normal allowed", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ext4", "rootcontext", "system_u:object_r:tmpfs_t:s0", true, nil)
		if len(issues) != 0 {
			t.Fatalf("expected 0 issues, got %d", len(issues))
		}
	})
}

func TestOptionalStringOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 5}
	opt := options.OptionalStringOption{}

	t.Run("allows empty value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "btrfs", "compress", "", false, nil)
		if len(issues) != 0 {
			t.Errorf("expected no issues, got %d", len(issues))
		}
	})

	t.Run("allows non-empty value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "btrfs", "compress", "zstd", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected no issues, got %d", len(issues))
		}
	})
}
