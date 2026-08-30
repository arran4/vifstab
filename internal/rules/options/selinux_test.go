package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestSELinuxContextOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 5, RawLine: "tmpfs /tmp tmpfs context=system_u:object_r:tmpfs_t:s0 0 0"}
	opt := options.SELinuxContextOption{}

	cases := []struct {
		name     string
		val      string
		hasVal   bool
		wantErr  bool
		errorMsg string
	}{
		{
			name:    "valid full context",
			val:     "system_u:object_r:tmpfs_t:s0",
			hasVal:  true,
			wantErr: false,
		},
		{
			name:    "valid partial context",
			val:     "system_u:object_r:tmpfs_t",
			hasVal:  true,
			wantErr: false,
		},
		{
			name:    "valid complex level context",
			val:     "unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023",
			hasVal:  true,
			wantErr: false,
		},
		{
			name:    "valid quoted context",
			val:     `"system_u:object_r:tmpfs_t:s0"`,
			hasVal:  true,
			wantErr: false,
		},
		{
			name:     "missing value",
			val:      "",
			hasVal:   false,
			wantErr:  true,
			errorMsg: "Missing value for tmpfs mount option 'context' on line 5",
		},
		{
			name:     "invalid format (too few parts)",
			val:      "system_u:object_r",
			hasVal:   true,
			wantErr:  true,
			errorMsg: "Invalid SELinux context format for tmpfs mount option 'context' on line 5: system_u:object_r",
		},
		{
			name:     "invalid format (spaces)",
			val:      "system_u: object_r: tmpfs_t: s0",
			hasVal:   true,
			wantErr:  true,
			errorMsg: "Invalid SELinux context format for tmpfs mount option 'context' on line 5: system_u: object_r: tmpfs_t: s0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "context", tc.val, tc.hasVal, nil)
			if tc.wantErr {
				if len(issues) == 0 {
					t.Fatalf("expected error, got none")
				}
				if issues[0].Message != tc.errorMsg {
					t.Errorf("expected error message %q, got %q", tc.errorMsg, issues[0].Message)
				}
			} else {
				if len(issues) > 0 {
					t.Errorf("expected no error, got %v", issues)
				}
			}
		})
	}
}

func TestSELinuxContextOption_Panic(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	entry := &rules.FstabEntry{LineNum: 5, RawLine: `tmpfs /tmp tmpfs context=" 0 0`}
	opt := options.SELinuxContextOption{}

	t.Run("single quote panic check", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "tmpfs", "context", `"`, true, nil)
		if len(issues) == 0 {
			t.Fatalf("expected error, got none")
		}
	})
}
