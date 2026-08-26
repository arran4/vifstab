package uefi

import (
	"strings"
	"testing"
	"github.com/arran4/vifstab/internal/rules"
)

func TestAnalyzeNil(t *testing.T) {
	// Not full coverage but making sure there's no obvious panic
	info, err := Analyze(rules.ActualFS{}, "UUID=test", "/boot")
	// If run on a non-UEFI system this returns nil, nil
	if err != nil {
		t.Log(err)
	}
	_ = info
}

func TestParseEFIPath(t *testing.T) {
	content := []byte{
		0, 0, 0, 0, // Attributes
		10, 0,       // FilePathListLength
		'g', 0, 'e', 0, 'n', 0, 't', 0, 'o', 0, 'o', 0, 0, 0, // Description "gentoo"
		4, 4, 18, 0, // Media Device Path / File Path, length 18
		'\\', 0, 'E', 0, 'F', 0, 'I', 0, '\\', 0, 'a', 0, 0, 0,
		0x7F, 0xFF, 4, 0,
	}

	desc, path := parseEFIPath(content)
	if desc != "gentoo" {
		t.Errorf("Expected description 'gentoo', got '%s'", desc)
	}
	if path != "\\EFI\\a" {
		t.Errorf("Expected path '\\EFI\\a', got '%s'", path)
	}
}

func TestFormatReport(t *testing.T) {
	info := &EFIInfo{
		DevicePath: "/dev/nvme1n1p6",
		UUID: "8F53-C3B9",
		PartUUID: "71cc0b59-9cd1-6642-9de4-d16d59f5ea94",
		IsESP: true,
		HasEFITree: true,
		NVRAMRefs: []NVRAMRef{
			{"gentoo", "\\EFI\\gentoo\\grubx64.efi"},
			{"Linux Boot Manager", "\\EFI\\systemd\\systemd-bootx64.efi"},
		},
		BootedMatch: true,
	}

	report := FormatReport(info, "/efi")
	expectedLines := []string{
		"/efi -> /dev/nvme1n1p6",
		"  UUID:     8F53-C3B9",
		"  PARTUUID: 71cc0b59-9cd1-6642-9de4-d16d59f5ea94",
		"  GPT role: EFI System Partition",
		"  EFI tree: present",
		"  NVRAM refs:",
		"    gentoo             \\EFI\\gentoo\\grubx64.efi",
		"    Linux Boot Manager \\EFI\\systemd\\systemd-bootx64.efi",
		"  current loader partition: match",
	}

	for _, line := range expectedLines {
		if !strings.Contains(report, line) {
			t.Errorf("Report missing line: %s\nReport was:\n%s", line, report)
		}
	}
}
