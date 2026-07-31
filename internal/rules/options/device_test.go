package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

func TestBlockDeviceOption(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{
		existingDevs: map[string]bool{
			"/dev/sdb1":                           true,
			"/dev/disk/by-uuid/1234-5678":         true,
			"/dev/disk/by-uuid/a1b2c3d4-5678":     true,
			"/dev/disk/by-label/BOOT":             true,
			"/dev/disk/by-label/My Disk":          true,
			"/dev/disk/by-partuuid/00001111-2222": true,
			"/dev/disk/by-partlabel/Home":         true,
		},
	}
	entry := &rules.FstabEntry{LineNum: 14, RawLine: "UUID=1234-5678 /mnt ext4 defaults 0 0"}
	opt := options.BlockDeviceOption{CheckExistence: true}

	t.Run("existing raw device path", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "logdev", "/dev/sdb1", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("existing UUID lookup", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "logdev", "UUID=1234-5678", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues for UUID lookup, got %d", len(issues))
		}
	})

	t.Run("existing quoted UUID lookup", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "logdev", `UUID="a1b2c3d4-5678"`, true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues for quoted UUID lookup, got %d", len(issues))
		}
	})

	t.Run("existing LABEL lookup", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "logdev", "LABEL=BOOT", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues for LABEL lookup, got %d", len(issues))
		}
	})

	t.Run("existing PARTUUID lookup", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "logdev", "PARTUUID=00001111-2222", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues for PARTUUID lookup, got %d", len(issues))
		}
	})

	t.Run("existing PARTLABEL lookup", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "logdev", "PARTLABEL=Home", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues for PARTLABEL lookup, got %d", len(issues))
		}
	})

	t.Run("escaped octal space in LABEL lookup", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "logdev", `LABEL=My\040Disk`, true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues for escaped LABEL lookup, got %d", len(issues))
		}
	})

	t.Run("non-existing device path", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "logdev", "/dev/sdz99", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Device '/dev/sdz99' specified in XFS mount option logdev does not exist on line 14"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})

	t.Run("non-existing UUID", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "logdev", "UUID=9999-9999", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Device 'UUID=9999-9999' specified in XFS mount option logdev does not exist on line 14"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
