package builtin_test

import (
	"context"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin"
)

type mockAPIPartition struct {
	devs []rules.BlockDevice
}

func (m mockAPIPartition) CheckFileExists(path string) rules.FileExistsResult { return rules.FileExists }
func (m mockAPIPartition) CheckUserExists(uid int) bool                       { return true }
func (m mockAPIPartition) CheckGroupExists(gid int) bool                      { return true }
func (m mockAPIPartition) GetHostKernelVersion() string                       { return "5.10.0" }
func (m mockAPIPartition) CheckBlockDeviceExists(dev string) bool             { return true }
func (m mockAPIPartition) GetBlockDevices() ([]rules.BlockDevice, error) {
	return m.devs, nil
}

func TestPartitionRoleRule(t *testing.T) {
	rule := &builtin.PartitionRoleRule{}
	api := mockAPIPartition{
		devs: []rules.BlockDevice{
			{
				Path:     "/dev/sda1",
				FSType:   "ext4",
				PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b",
				Label:    "gentooboot",
			},
			{
				Path:     "/dev/sda2",
				FSType:   "vfat",
				PartType: "ebd0a0a2-b9e5-4433-87c0-68b6b72699c7",
				UUID:     "8F53-C3B9",
				MountPoints: []string{"/efi"},
			},
		},
	}

	ctx := context.Background()

	// Test case 1: /boot with ESP type but ext4
	entry1 := &rules.FstabEntry{
		RawLine: "LABEL=gentooboot /boot ext4 defaults 1 2",
		LineNum: 1,
		Fields:  []string{"LABEL=gentooboot", "/boot", "ext4", "defaults", "1", "2"},
	}
	issues1, _ := rule.Validate(ctx, api, entry1)
	if len(issues1) == 0 {
		t.Errorf("Expected issue for /boot but got none")
	} else if issues1[0].Message != "GPT partition type is EFI System Partition (ESP) but filesystem is not FAT" {
		t.Errorf("Unexpected message: %s", issues1[0].Message)
	}

	// Test case 2: /efi with non-ESP type and vfat
	entry2 := &rules.FstabEntry{
		RawLine: "UUID=8F53-C3B9 /efi vfat defaults 1 2",
		LineNum: 2,
		Fields:  []string{"UUID=8F53-C3B9", "/efi", "vfat", "defaults", "1", "2"},
	}
	issues2, _ := rule.Validate(ctx, api, entry2)
	if len(issues2) == 0 {
		t.Errorf("Expected issue for /efi but got none")
	} else if issues2[0].Message != "FAT filesystem mounted at EFI mountpoint but GPT partition type is not ESP" {
		t.Errorf("Unexpected message: %s", issues2[0].Message)
	}

	// Test case 3: missing block devices gracefully ignored
	apiEmpty := mockAPIPartition{devs: nil}
	issues3, _ := rule.Validate(ctx, apiEmpty, entry1)
	if len(issues3) != 0 {
		t.Errorf("Expected no issues when block device is missing, got %d", len(issues3))
	}
}
