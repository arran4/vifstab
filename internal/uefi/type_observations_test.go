package uefi

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
)

func TestIncompleteTypeMetadataIsNotNonESP(t *testing.T) {
	for _, tc := range []struct {
		name, partType, role string
	}{
		{"missing", "", "unknown"},
		{"malformed", "not-a-guid", "unknown"},
		{"invalid-hex", "z12a7328-f81f-11d2-ba4b-00a0c93ec93b", "unknown"},
		{"misplaced-separators", "c12a7328f-81f-11d2-ba4b-00a0c93ec93b", "unknown"},
		{"esp", "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", "EFI System Partition"},
		{"uppercase-esp", "C12A7328-F81F-11D2-BA4B-00A0C93EC93B", "EFI System Partition"},
		{"other-type", "0fc63daf-8483-4772-8e79-3d69d8477de4", "Not ESP"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := mockAPI{devices: []rules.BlockDevice{{Path: "/dev/fixture", PartType: tc.partType}}}
			provider := mockEFIProv{statFunc: func(name string) (fs.FileInfo, error) {
				if name != "/sys/firmware/efi" {
					t.Fatalf("unexpected mounted-tree access: %s", name)
				}
				return mockFileInfo{}, nil
			}}
			info, err := Analyze(api, provider, "/dev/fixture", "")
			if err != nil || info == nil {
				t.Fatalf("analysis = %#v, %v", info, err)
			}
			report := FormatReport(info, "/dev/fixture")
			if !strings.Contains(report, "  GPT role: "+tc.role+"\n") {
				t.Fatalf("missing expected role %q: %s", tc.role, report)
			}
			if (info.IsESP == nil) != (tc.role == "unknown") {
				t.Fatalf("role observation lost its unknown state: %#v", info.IsESP)
			}
		})
	}
}

func TestFilesystemTypeObservationDistinguishesUnknown(t *testing.T) {
	for _, tc := range []struct {
		fstype     string
		known, fat bool
	}{{"", false, false}, {" ", false, false}, {"vfat", true, true}, {"fat", true, true}, {"msdos", true, true}, {"ext4", true, false}} {
		api := mockAPI{devices: []rules.BlockDevice{{Path: "/dev/fixture", FSType: tc.fstype}}}
		provider := mockEFIProv{statFunc: func(string) (fs.FileInfo, error) { return mockFileInfo{}, nil }}
		info, err := Analyze(api, provider, "/dev/fixture", "")
		if err != nil || info == nil {
			t.Fatalf("analysis = %#v, %v", info, err)
		}
		if (info.IsFAT != nil) != tc.known || (info.IsFAT != nil && *info.IsFAT != tc.fat) {
			t.Fatalf("filesystem %q observation = %v, want known=%v fat=%v", tc.fstype, info.IsFAT, tc.known, tc.fat)
		}
	}
}
