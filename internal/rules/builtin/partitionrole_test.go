package builtin_test

import (
	"context"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin"
)

type mockAPIPartition struct {
	devs []rules.BlockDevice
}

func (m mockAPIPartition) CheckFileExists(path string) rules.FileExistsResult {
	return rules.FileExists
}
func (m mockAPIPartition) CheckUserExists(uid int) bool           { return true }
func (m mockAPIPartition) CheckGroupExists(gid int) bool          { return true }
func (m mockAPIPartition) GetHostKernelVersion() string           { return "5.10.0" }
func (m mockAPIPartition) CheckBlockDeviceExists(dev string) bool { return true }
func (m mockAPIPartition) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	return rules.BlockDeviceInventory{Devices: m.devs, Available: true}, nil
}
func (m mockAPIPartition) ResolveDevicePath(dev string) string {
	if strings.HasPrefix(dev, "LABEL=") {
		for _, d := range m.devs {
			if d.Label == strings.TrimPrefix(dev, "LABEL=") {
				return d.Path
			}
		}
	}
	if strings.HasPrefix(dev, "UUID=") {
		for _, d := range m.devs {
			if d.UUID == strings.TrimPrefix(dev, "UUID=") {
				return d.Path
			}
		}
	}
	if strings.HasPrefix(dev, "PARTUUID=") {
		for _, d := range m.devs {
			if d.PartUUID == strings.TrimPrefix(dev, "PARTUUID=") {
				return d.Path
			}
		}
	}
	if strings.HasPrefix(dev, "/dev/disk/by-uuid/") {
		for _, d := range m.devs {
			if d.UUID == strings.TrimPrefix(dev, "/dev/disk/by-uuid/") {
				return d.Path
			}
		}
	}
	return dev
}

func TestPartitionRoleRule(t *testing.T) {
	rule := &builtin.PartitionRoleRule{}

	cases := []struct {
		name       string
		fstabEntry *rules.FstabEntry
		api        mockAPIPartition
		wantIssues []string
	}{
		{
			name: "declared vfat, detected ext4, ESP",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot/efi vfat defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot/efi", "vfat", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{Path: "/dev/sda1", FSType: "ext4", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", Label: "gentooboot"},
				},
			},
			wantIssues: []string{
				"GPT partition type is EFI System Partition (ESP) but filesystem is not FAT",
				"Declared filesystem type does not match detected live filesystem type",
			},
		},
		{
			name: "declared ext4, detected vfat, ESP",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot/efi ext4 defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot/efi", "ext4", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{Path: "/dev/sda1", FSType: "vfat", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", Label: "gentooboot"},
				},
			},
			wantIssues: []string{
				"Declared filesystem type does not match detected live filesystem type",
			},
		},
		{
			name: "declared auto, detected vfat",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot/efi auto defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot/efi", "auto", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{Path: "/dev/sda1", FSType: "vfat", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", Label: "gentooboot"},
				},
			},
			wantIssues: []string{}, // No mismatch
		},
		{
			name: "declared auto, detected ext4, ESP",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot/efi auto defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot/efi", "auto", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{Path: "/dev/sda1", FSType: "ext4", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", Label: "gentooboot"},
				},
			},
			wantIssues: []string{
				"GPT partition type is EFI System Partition (ESP) but filesystem is not FAT",
			}, // Mismatch not reported on auto, but ESP type requires FAT
		},
		{
			name: "valid declared/detected vfat + ESP",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot/efi vfat defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot/efi", "vfat", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{Path: "/dev/sda1", FSType: "vfat", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", Label: "gentooboot"},
				},
			},
			wantIssues: []string{},
		},
		{
			name: "actual non-FAT + ESP",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot/efi ext4 defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot/efi", "ext4", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{Path: "/dev/sda1", FSType: "ext4", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", Label: "gentooboot"},
				},
			},
			wantIssues: []string{
				"GPT partition type is EFI System Partition (ESP) but filesystem is not FAT",
			},
		},
		{
			name: "detected filesystem unavailable",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot/efi ext4 defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot/efi", "ext4", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{Path: "/dev/sda1", FSType: "", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", Label: "gentooboot"},
				},
			},
			wantIssues: []string{}, // do not guess from fstab
		},
		{
			name: "FAT EFI mount with non-ESP type",
			fstabEntry: &rules.FstabEntry{
				RawLine: "UUID=8F53-C3B9 /efi vfat defaults 1 2",
				LineNum: 2,
				Fields:  []string{"UUID=8F53-C3B9", "/efi", "vfat", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{Path: "/dev/sda2", FSType: "vfat", PartType: "ebd0a0a2-b9e5-4433-87c0-68b6b72699c7", UUID: "8F53-C3B9"},
				},
			},
			wantIssues: []string{
				"FAT filesystem mounted at EFI mountpoint but GPT partition type is not ESP",
			},
		},
		{
			name: "/boot ESP plus genuinely distinct FAT EFI device",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot ext4 defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot", "ext4", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{DeviceID: "8:1", Path: "/dev/sda1", FSType: "ext4", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", Label: "gentooboot"},
					{DeviceID: "8:2", Path: "/dev/sda2", FSType: "vfat", PartType: "ebd0a0a2-b9e5-4433-87c0-68b6b72699c7", UUID: "8F53-C3B9", MountPoints: []string{"/efi"}},
				},
			},
			wantIssues: []string{
				"GPT partition type is EFI System Partition (ESP) but filesystem is not FAT",
				"/boot is typed as ESP but a distinct FAT EFI partition is present",
			},
		},
		{
			name: "same physical device represented through multiple aliases/mountpoints",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot ext4 defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot", "ext4", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{DeviceID: "8:1", Path: "/dev/sda1", FSType: "vfat", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", Label: "gentooboot", MountPoints: []string{"/efi"}},
					{DeviceID: "8:1", Path: "/dev/disk/by-uuid/123", FSType: "vfat", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", Label: "gentooboot", MountPoints: []string{"/efi"}}, // same DeviceID!
				},
			},
			wantIssues: []string{
				"Declared filesystem type does not match detected live filesystem type",
			}, // Should not warn about distinct EFI
		},
		{
			name: "UUID resolution",
			fstabEntry: &rules.FstabEntry{
				RawLine: "UUID=8F53-C3B9 /boot/efi vfat defaults 1 2",
				LineNum: 1,
				Fields:  []string{"UUID=8F53-C3B9", "/boot/efi", "vfat", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{Path: "/dev/sda1", FSType: "vfat", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", UUID: "8F53-C3B9"},
				},
			},
			wantIssues: []string{},
		},
		{
			name: "/dev/disk/by-uuid resolution",
			fstabEntry: &rules.FstabEntry{
				RawLine: "/dev/disk/by-uuid/8F53-C3B9 /boot/efi vfat defaults 1 2",
				LineNum: 1,
				Fields:  []string{"/dev/disk/by-uuid/8F53-C3B9", "/boot/efi", "vfat", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{Path: "/dev/sda1", FSType: "vfat", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", UUID: "8F53-C3B9"},
				},
			},
			wantIssues: []string{},
		},
		{
			name: "missing device",
			fstabEntry: &rules.FstabEntry{
				RawLine: "UUID=8F53-C3B9 /boot/efi vfat defaults 1 2",
				LineNum: 1,
				Fields:  []string{"UUID=8F53-C3B9", "/boot/efi", "vfat", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{},
			},
			wantIssues: []string{
				"Source could not be resolved against current live block metadata and partition-role check was skipped",
			}, // graceful ignore with info message
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			issues, _ := rule.Validate(ctx, tc.api, tc.fstabEntry)

			if len(issues) != len(tc.wantIssues) {
				t.Fatalf("Got %d issues, want %d.\nGot: %v\nWant: %v", len(issues), len(tc.wantIssues), issues, tc.wantIssues)
			}

			for i, issue := range issues {
				if issue.Message != tc.wantIssues[i] {
					t.Errorf("Issue %d message got = %q, want = %q", i, issue.Message, tc.wantIssues[i])
				}
			}
		})
	}
}
