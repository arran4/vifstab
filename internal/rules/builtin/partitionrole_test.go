package builtin_test

import (
	"context"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/rules/builtin"
	"github.com/arran4/vifstab/internal/rules/core"
)

type mockAPIPartition struct {
	devs      []rules.BlockDevice
	available bool
}

func (m mockAPIPartition) CheckFileExists(path string) rules.FileExistsResult {
	return rules.FileExists
}

func (m mockAPIPartition) CheckBlockDeviceExists(dev string) bool {
	return true
}

func (m mockAPIPartition) GetHostKernelVersion() string {
	return ""
}

func (m mockAPIPartition) CheckUserExists(uid int) bool {
	return true
}

func (m mockAPIPartition) CheckGroupExists(gid int) bool {
	return true
}

func (m mockAPIPartition) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	return rules.BlockDeviceInventory{
		Devices:   m.devs,
		Available: m.available,
	}, nil
}

func (m mockAPIPartition) ResolveDevicePath(dev string) string {
	return rules.ResolveDevicePath(dev)
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
			name: "Gentoo /boot ESP with ext4",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot ext4 defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot", "ext4", "defaults", "1", "2"},
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
			name: "Gentoo /efi with non-ESP type",
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
			name: "mismatched declared fstype vs detected",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot ext4 defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot", "ext4", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs: []rules.BlockDevice{
					{Path: "/dev/sda1", FSType: "btrfs", PartType: "0fc63daf-8483-4772-8e79-3d69d8477de4", Label: "gentooboot"},
				},
			},
			wantIssues: []string{
				"Declared filesystem type does not match detected live filesystem type",
			},
		},
		{
			name: "auto fstype ignores declared mismatch",
			fstabEntry: &rules.FstabEntry{
				RawLine: "LABEL=gentooboot /boot auto defaults 1 2",
				LineNum: 1,
				Fields:  []string{"LABEL=gentooboot", "/boot", "auto", "defaults", "1", "2"},
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
			name: "missing device (available=true)",
			fstabEntry: &rules.FstabEntry{
				RawLine: "UUID=8F53-C3B9 /boot/efi vfat defaults 1 2",
				LineNum: 1,
				Fields:  []string{"UUID=8F53-C3B9", "/boot/efi", "vfat", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs:      []rules.BlockDevice{},
				available: true,
			},
			wantIssues: []string{
				"Source could not be resolved against current live block metadata and partition-role check was skipped",
			}, // graceful ignore with info message
		},
		{
			name: "missing device (available=false)",
			fstabEntry: &rules.FstabEntry{
				RawLine: "UUID=8F53-C3B9 /boot/efi vfat defaults 1 2",
				LineNum: 1,
				Fields:  []string{"UUID=8F53-C3B9", "/boot/efi", "vfat", "defaults", "1", "2"},
			},
			api: mockAPIPartition{
				devs:      []rules.BlockDevice{},
				available: false,
			},
			wantIssues: []string{
				"Live metadata check unavailable for this block device",
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

func TestPartitionRoleRule_MoreResolution(t *testing.T) {
	rule := &builtin.PartitionRoleRule{}
	ctx := context.Background()

	api := mockAPIPartition{
		available: true,
		devs: []rules.BlockDevice{
			{
				Path:     "/dev/sda1",
				FSType:   "vfat",
				PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b",
				Label:    "Gentoo Boot",
				PartUUID: "1234",
				Aliases: []string{
					"/dev/disk/by-label/gentooboot",
					"/dev/disk/by-partuuid/1234",
					"/dev/disk/by-path/pci-0000:00:1f.2-ata-1",
					"/dev/disk/by-id/ata-VBOX_HARDDISK_VB0e7f7881-80718d05",
					"/dev/disk/by-partlabel/EFI\\x20System\\x20Partition",
				},
			},
		},
	}

	validEntries := []string{
		"PARTUUID=1234 /boot/efi vfat defaults 1 2",
		"LABEL=Gentoo\\040Boot /boot/efi vfat defaults 1 2",
		"/dev/disk/by-partuuid/1234 /boot/efi vfat defaults 1 2",
		"/dev/sda1 /boot/efi vfat defaults 1 2",
		"/dev/disk/by-path/pci-0000:00:1f.2-ata-1 /boot/efi vfat defaults 1 2",
		"/dev/disk/by-id/ata-VBOX_HARDDISK_VB0e7f7881-80718d05 /boot/efi vfat defaults 1 2",
		"/dev/disk/by-partlabel/EFI\\040System\\040Partition /boot/efi vfat defaults 1 2",
		"PARTLABEL=EFI\\040System\\040Partition /boot/efi vfat defaults 1 2",
		"ID=ata-VBOX_HARDDISK_VB0e7f7881-80718d05 /boot/efi vfat defaults 1 2",
	}

	for _, line := range validEntries {
		entries, err := core.ParseFstab(strings.NewReader(line))
		if err != nil {
			t.Fatalf("ParseFstab(%q): %v", line, err)
		}

		issues, _ := rule.Validate(ctx, api, entries[0])
		if len(issues) != 0 {
			t.Errorf("expected 0 issues for %s, got %v", line, issues)
		}
	}
}
