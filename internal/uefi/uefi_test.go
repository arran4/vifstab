package uefi

import (
	"encoding/binary"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/arran4/vifstab/internal/rules"
)

type mockEFIProv struct {
	statFunc     func(name string) (fs.FileInfo, error)
	readDirFunc  func(name string) ([]fs.DirEntry, error)
	readFileFunc func(name string) ([]byte, error)
}

func (m mockEFIProv) Stat(name string) (fs.FileInfo, error) {
	if m.statFunc != nil {
		return m.statFunc(name)
	}
	return nil, osErrNotExist
}

func (m mockEFIProv) ReadDir(name string) ([]fs.DirEntry, error) {
	if m.readDirFunc != nil {
		return m.readDirFunc(name)
	}
	return nil, osErrNotExist
}

func (m mockEFIProv) ReadFile(name string) ([]byte, error) {
	if m.readFileFunc != nil {
		return m.readFileFunc(name)
	}
	return nil, osErrNotExist
}

var osErrNotExist = fs.ErrNotExist

type mockDirEntry struct {
	name string
}

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                { return false }
func (m mockDirEntry) Type() fs.FileMode          { return 0 }
func (m mockDirEntry) Info() (fs.FileInfo, error) { return mockFileInfo{name: m.name}, nil }

type mockFileInfo struct {
	name string
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return 0 }
func (m mockFileInfo) Mode() fs.FileMode  { return 0 }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool        { return false }
func (m mockFileInfo) Sys() any           { return nil }

func buildLoadOption(desc string, filePath string, partUUID []byte, valid bool) []byte {
	// Attributes (4 bytes)
	out := []byte{0, 0, 0, 0}

	// We will fill FilePathListLength later
	out = append(out, 0, 0)

	// Description
	for _, r := range desc {
		out = append(out, byte(r), 0)
	}
	out = append(out, 0, 0) // Null terminator

	listStart := len(out)

	// Hard Drive Media Device Path
	// Type 4, SubType 1, Length 42
	out = append(out, 4, 1, 42, 0)

	// Fill the rest with 0s up to 42 bytes
	hdPath := make([]byte, 38)
	// Offset 20 within hdPath is offset 24 within the whole node.
	if partUUID != nil && len(partUUID) == 16 {
		copy(hdPath[20:], partUUID)
	}
	// Partition Format = 0x02, Signature Type = 0x02
	hdPath[36] = 0x02
	hdPath[37] = 0x02
	out = append(out, hdPath...)

	// File Path
	if filePath != "" {
		filePathRunes := []rune(filePath)
		fpLength := 4 + len(filePathRunes)*2 + 2
		out = append(out, 4, 4, byte(fpLength), byte(fpLength>>8))
		for _, r := range filePathRunes {
			out = append(out, byte(r), 0)
		}
		out = append(out, 0, 0)
	}

	// End of list
	if valid {
		out = append(out, 0x7F, 0xFF, 4, 0)
	}

	listLen := len(out) - listStart
	binary.LittleEndian.PutUint16(out[4:6], uint16(listLen))

	// Prepend the 4 bytes of EFIVAR attributes
	finalOut := []byte{7, 0, 0, 0}
	finalOut = append(finalOut, out...)
	return finalOut
}

type mockAPI struct {
	rules.API
	devices []rules.BlockDevice
}

func (m mockAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	return rules.BlockDeviceInventory{
		Devices: m.devices,
		Available: true,
	}, nil
}

func TestAnalyze_HappyPath(t *testing.T) {
	partUUID := "71cc0b59-9cd1-6642-9de4-d16d59f5ea94"
	partUUIDBytes, _ := parseUUID(partUUID)

	api := mockAPI{
		devices: []rules.BlockDevice{
			{
				Path:        "/dev/nvme1n1p6",
				UUID:        "8F53-C3B9",
				PartUUID:    partUUID,
				FSType:      "vfat",
				PartType:    "c12a7328-f81f-11d2-ba4b-00a0c93ec93b",
				MountPoints: []string{"/efi"},
			},
		},
	}

	prov := mockEFIProv{
		statFunc: func(name string) (fs.FileInfo, error) {
			return mockFileInfo{}, nil
		},
		readDirFunc: func(name string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{
				mockDirEntry{name: "Boot0001-8be4df61-93ca-11d2-aa0d-00e098032b8c"},
				mockDirEntry{name: "Boot0002-8be4df61-93ca-11d2-aa0d-00e098032b8c"},
			}, nil
		},
		readFileFunc: func(name string) ([]byte, error) {
			if strings.Contains(name, "BootCurrent") {
				// Current is 0001
				out := []byte{7, 0, 0, 0, 1, 0}
				return out, nil
			}
			if strings.Contains(name, "Boot0001") {
				return buildLoadOption("gentoo", "\\EFI\\gentoo\\grubx64.efi", partUUIDBytes, true), nil
			}
			if strings.Contains(name, "Boot0002") {
				otherUUID, _ := parseUUID("11111111-2222-3333-4444-555555555555")
				return buildLoadOption("Windows", "\\EFI\\Microsoft\\Boot\\bootmgfw.efi", otherUUID, true), nil
			}
			return nil, osErrNotExist
		},
	}

	info, err := Analyze(api, prov, "/dev/nvme1n1p6", "/efi")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if info == nil {
		t.Fatalf("info is nil")
	}
	if !info.IsESP {
		t.Errorf("Expected IsESP true")
	}
	if info.HasEFITree == nil || !*info.HasEFITree {
		t.Errorf("Expected HasEFITree true")
	}
	if info.NVRAMRefs == nil || len(*info.NVRAMRefs) != 1 {
		t.Fatalf("Expected 1 NVRAM ref, got %v", info.NVRAMRefs)
	}
	if (*info.NVRAMRefs)[0].Name != "gentoo" {
		t.Errorf("Expected 'gentoo', got '%s'", (*info.NVRAMRefs)[0].Name)
	}
	if info.BootedMatch == nil || !*info.BootedMatch {
		t.Errorf("Expected BootedMatch true")
	}
}

func TestAnalyze_UnknownBootCurrent(t *testing.T) {
	partUUID := "71cc0b59-9cd1-6642-9de4-d16d59f5ea94"
	partUUIDBytes, _ := parseUUID(partUUID)

	api := mockAPI{
		devices: []rules.BlockDevice{
			{
				Path:        "/dev/nvme1n1p6",
				UUID:        "8F53-C3B9",
				PartUUID:    partUUID,
				FSType:      "vfat",
				PartType:    "c12a7328-f81f-11d2-ba4b-00a0c93ec93b",
				MountPoints: []string{"/efi"},
			},
		},
	}

	prov := mockEFIProv{
		statFunc: func(name string) (fs.FileInfo, error) {
			return mockFileInfo{}, nil
		},
		readDirFunc: func(name string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{
				mockDirEntry{name: "Boot0001-8be4df61-93ca-11d2-aa0d-00e098032b8c"},
			}, nil
		},
		readFileFunc: func(name string) ([]byte, error) {
			if strings.Contains(name, "BootCurrent") {
				// Malformed BootCurrent
				return []byte{7, 0, 0, 0}, nil
			}
			if strings.Contains(name, "Boot0001") {
				return buildLoadOption("gentoo", "\\EFI\\gentoo\\grubx64.efi", partUUIDBytes, true), nil
			}
			return nil, osErrNotExist
		},
	}

	info, err := Analyze(api, prov, "/dev/nvme1n1p6", "/efi")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if info.BootedMatch != nil {
		t.Errorf("Expected BootedMatch nil (unknown), got %v", *info.BootedMatch)
	}
}

func TestAnalyze_MismatchBootCurrent(t *testing.T) {
	partUUID := "71cc0b59-9cd1-6642-9de4-d16d59f5ea94"
	partUUIDBytes, _ := parseUUID(partUUID)
	otherUUID, _ := parseUUID("11111111-2222-3333-4444-555555555555")

	api := mockAPI{
		devices: []rules.BlockDevice{
			{
				Path:        "/dev/nvme1n1p6",
				UUID:        "8F53-C3B9",
				PartUUID:    partUUID,
				FSType:      "vfat",
				PartType:    "c12a7328-f81f-11d2-ba4b-00a0c93ec93b",
				MountPoints: []string{"/efi"},
			},
		},
	}

	prov := mockEFIProv{
		statFunc: func(name string) (fs.FileInfo, error) {
			return mockFileInfo{}, nil
		},
		readDirFunc: func(name string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{
				mockDirEntry{name: "Boot0001-8be4df61-93ca-11d2-aa0d-00e098032b8c"},
				mockDirEntry{name: "Boot0002-8be4df61-93ca-11d2-aa0d-00e098032b8c"},
			}, nil
		},
		readFileFunc: func(name string) ([]byte, error) {
			if strings.Contains(name, "BootCurrent") {
				// Current is 0002
				out := []byte{7, 0, 0, 0, 2, 0}
				return out, nil
			}
			if strings.Contains(name, "Boot0001") {
				return buildLoadOption("gentoo", "\\EFI\\gentoo\\grubx64.efi", partUUIDBytes, true), nil
			}
			if strings.Contains(name, "Boot0002") {
				return buildLoadOption("Windows", "\\EFI\\Microsoft\\Boot\\bootmgfw.efi", otherUUID, true), nil
			}
			return nil, osErrNotExist
		},
	}

	info, err := Analyze(api, prov, "/dev/nvme1n1p6", "/efi")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if info.BootedMatch == nil {
		t.Errorf("Expected BootedMatch to be false, got nil")
	} else if *info.BootedMatch {
		t.Errorf("Expected BootedMatch false")
	}
}


type customErr struct { msg string }
func (e customErr) Error() string { return e.msg }

func TestAnalyze_PermissionDeniedEFI(t *testing.T) {
	partUUID := "71cc0b59-9cd1-6642-9de4-d16d59f5ea94"
	api := mockAPI{
		devices: []rules.BlockDevice{
			{
				Path:        "/dev/nvme1n1p6",
				UUID:        "8F53-C3B9",
				PartUUID:    partUUID,
				FSType:      "vfat",
				PartType:    "c12a7328-f81f-11d2-ba4b-00a0c93ec93b",
				MountPoints: []string{"/efi"},
			},
		},
	}

	prov := mockEFIProv{
		statFunc: func(name string) (fs.FileInfo, error) {
			if strings.Contains(name, "EFI") {
				return nil, customErr{"permission denied"}
			}
			return mockFileInfo{}, nil
		},
		readDirFunc: func(name string) ([]fs.DirEntry, error) {
			return nil, osErrNotExist
		},
	}

	info, err := Analyze(api, prov, "/dev/nvme1n1p6", "/efi")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if info.HasEFITree != nil {
		t.Errorf("Expected HasEFITree nil (unknown), got %v", *info.HasEFITree)
	}
}

func TestAnalyze_UnreadableBootEntry(t *testing.T) {
	partUUID := "71cc0b59-9cd1-6642-9de4-d16d59f5ea94"
	api := mockAPI{
		devices: []rules.BlockDevice{
			{
				Path:        "/dev/nvme1n1p6",
				UUID:        "8F53-C3B9",
				PartUUID:    partUUID,
				FSType:      "vfat",
				PartType:    "c12a7328-f81f-11d2-ba4b-00a0c93ec93b",
				MountPoints: []string{"/efi"},
			},
		},
	}

	prov := mockEFIProv{
		statFunc: func(name string) (fs.FileInfo, error) {
			return mockFileInfo{}, nil
		},
		readDirFunc: func(name string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{
				mockDirEntry{name: "Boot0001-8be4df61-93ca-11d2-aa0d-00e098032b8c"},
				mockDirEntry{name: "Boot0002-8be4df61-93ca-11d2-aa0d-00e098032b8c"},
			}, nil
		},
		readFileFunc: func(name string) ([]byte, error) {
			if strings.Contains(name, "BootCurrent") {
				return nil, osErrNotExist
			}
			if strings.Contains(name, "Boot0001") {
				otherUUID, _ := parseUUID("11111111-2222-3333-4444-555555555555")
				return buildLoadOption("Windows", "\\EFI\\Microsoft\\Boot\\bootmgfw.efi", otherUUID, true), nil
			}
			if strings.Contains(name, "Boot0002") {
				return nil, customErr{"permission denied"}
			}
			return nil, osErrNotExist
		},
	}

	info, err := Analyze(api, prov, "/dev/nvme1n1p6", "/efi")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if info.NVRAMRefs != nil {
		t.Errorf("Expected NVRAMRefs nil (unknown), got %v", info.NVRAMRefs)
	}
}
