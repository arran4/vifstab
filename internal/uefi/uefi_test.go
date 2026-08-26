package uefi

import (
	"strings"
	"testing"
	"github.com/arran4/vifstab/internal/rules"
)

func TestAnalyzeNil(t *testing.T) {
	info, err := Analyze(rules.ActualFS{}, "UUID=test", "/boot")
	if err != nil {
		t.Log(err)
	}
	_ = info
}

func TestParseEFIPath(t *testing.T) {
	content := []byte{
		0, 0, 0, 0, // Attributes
		18, 0,       // FilePathListLength (corrected)
		'g', 0, 'e', 0, 'n', 0, 't', 0, 'o', 0, 'o', 0, 0, 0, // Description "gentoo" (14 bytes)
		4, 4, 18, 0, // Media Device Path / File Path, length 18
		'\\', 0, 'E', 0, 'F', 0, 'I', 0, '\\', 0, 'a', 0, 0, 0,
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
    bTrue := true
    bFalse := false
	info := &EFIInfo{
		DevicePath: "/dev/nvme1n1p6",
		UUID: "8F53-C3B9",
		PartUUID: "71cc0b59-9cd1-6642-9de4-d16d59f5ea94",
		IsESP: true,
		HasEFITree: &bTrue,
		NVRAMRefs: &[]NVRAMRef{
			{"gentoo", "\\EFI\\gentoo\\grubx64.efi"},
			{"Linux Boot Manager", "\\EFI\\systemd\\systemd-bootx64.efi"},
		},
		BootedMatch: &bTrue,
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

    info.HasEFITree = nil
    info.NVRAMRefs = nil
    info.BootedMatch = &bFalse

    report2 := FormatReport(info, "/efi")
    if !strings.Contains(report2, "EFI tree: unknown") {
        t.Errorf("Expected unknown tree")
    }
    if !strings.Contains(report2, "NVRAM refs:\n    (unknown)") {
        t.Errorf("Expected unknown refs")
    }
    if !strings.Contains(report2, "current loader partition: mismatch") {
        t.Errorf("Expected mismatch")
    }
}
