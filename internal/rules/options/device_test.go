package options_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/options"
)

type mockDeviceAPI struct {
	mockAPI
	existingDevs map[string]bool
}

func (m *mockDeviceAPI) CheckBlockDeviceExists(dev string) bool {
	if m.existingDevs == nil {
		return false
	}
	return m.existingDevs[dev]
}

func TestBlockDeviceOption(t *testing.T) {
	ctx := context.Background()
	api := &mockDeviceAPI{
		existingDevs: map[string]bool{"/dev/sdb1": true},
	}
	entry := &rules.FstabEntry{LineNum: 14}
	opt := options.BlockDeviceOption{CheckExistence: true}

	t.Run("existing device", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "logdev", "/dev/sdb1", true, nil)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues, got %d", len(issues))
		}
	})

	t.Run("non-existing device", func(t *testing.T) {
		issues := opt.ValidateOption(ctx, api, entry, "XFS", "logdev", "/dev/sdz99", true, nil)
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		expectedMsg := "Device '/dev/sdz99' specified in XFS mount option logdev does not exist on line 14"
		if issues[0].Message != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, issues[0].Message)
		}
	})
}
