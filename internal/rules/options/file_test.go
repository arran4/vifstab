package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestFileOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{
		existingFiles: map[string]bool{
			"/etc/ceph/admin.secret": true,
			"/etc/ceph/my.conf":      true,
			"/etc/ceph/escaped file": true,
		},
	}
	entry := &rules.FstabEntry{LineNum: 14, RawLine: "192.168.0.1:/ /mnt ceph name=admin,secretfile=/etc/ceph/admin.secret 0 0"}
	opt := options.FileOption{CheckExistence: true}

	t.Run("existing file", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ceph", "secretfile", "/etc/ceph/admin.secret", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("existing escaped file", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ceph", "secretfile", `/etc/ceph/escaped\040file`, true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("non-existing file", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ceph", "secretfile", "/etc/ceph/missing.secret", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "File '/etc/ceph/missing.secret' specified in ceph mount option 'secretfile' does not exist on line 14"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("missing value", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "ceph", "secretfile", "", false, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Missing file path for ceph mount option 'secretfile' on line 14"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
