package editor

import (
	"errors"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
)

type countingAPI struct {
	rules.API
	getBlockDevicesCount int
	checkFileExistsCount int
	invToReturn          rules.BlockDeviceInventory
	errToReturn          error
}

func (c *countingAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	c.getBlockDevicesCount++
	return c.invToReturn, c.errToReturn
}

func (c *countingAPI) CheckFileExists(path string) rules.FileExistsResult {
	c.checkFileExistsCount++
	return rules.FileExists
}

func (c *countingAPI) CheckBlockDeviceExists(dev string) bool {
	return true
}

func (c *countingAPI) GetHostKernelVersion() string {
	return "6.6.0"
}

func (c *countingAPI) CheckUserExists(uid int) bool {
	return true
}

func (c *countingAPI) CheckGroupExists(gid int) bool {
	return true
}

func (c *countingAPI) ResolveDevicePath(dev string) string {
	return rules.ResolveDevicePath(dev)
}

func TestMemoizedAPI_GetBlockDevices(t *testing.T) {
	expectedErr := errors.New("test inventory error")
	expectedInv := rules.BlockDeviceInventory{
		Devices: []rules.BlockDevice{
			{Path: "/dev/sda1", FSType: "vfat", UUID: "1111-2222"},
		},
		Diagnostics: []rules.BlockDeviceDiagnostic{
			{Source: "test", Error: "diag error"},
		},
		Available: true,
	}

	api := &countingAPI{
		invToReturn: expectedInv,
		errToReturn: expectedErr,
	}
	memoized := &memoizedAPI{API: api}

	inv1, err1 := memoized.GetBlockDevices()
	inv2, err2 := memoized.GetBlockDevices()

	if api.getBlockDevicesCount != 1 {
		t.Errorf("GetBlockDevices called %d times, expected 1", api.getBlockDevicesCount)
	}
	if err1 != expectedErr || err2 != expectedErr {
		t.Errorf("expected error %v, got err1=%v, err2=%v", expectedErr, err1, err2)
	}
	if !inv1.Available || !inv2.Available {
		t.Errorf("expected Available=true, got inv1=%v, inv2=%v", inv1.Available, inv2.Available)
	}
	if len(inv1.Devices) != 1 || len(inv2.Devices) != 1 {
		t.Errorf("expected 1 device, got %d and %d", len(inv1.Devices), len(inv2.Devices))
	}
	if len(inv1.Diagnostics) != 1 || inv1.Diagnostics[0].Source != "test" {
		t.Errorf("expected diagnostics preserved, got %v", inv1.Diagnostics)
	}
}

func TestMemoizedAPI_Delegation(t *testing.T) {
	api := &countingAPI{}
	memoized := &memoizedAPI{API: api}

	_ = memoized.CheckFileExists("/etc/fstab")
	if api.checkFileExistsCount != 1 {
		t.Errorf("CheckFileExists called %d times on underlying API, expected 1", api.checkFileExistsCount)
	}

	if memoized.GetHostKernelVersion() != "6.6.0" {
		t.Errorf("GetHostKernelVersion did not delegate properly")
	}

	if !memoized.CheckUserExists(1000) || !memoized.CheckGroupExists(1000) || !memoized.CheckBlockDeviceExists("/dev/sda1") {
		t.Errorf("delegated methods failed")
	}

	if memoized.ResolveDevicePath("UUID=123") != "/dev/disk/by-uuid/123" {
		t.Errorf("ResolveDevicePath did not delegate properly")
	}
}

func TestLintEntries_MemoizationIntegration_MultipleRelevantEntries(t *testing.T) {
	api := &countingAPI{
		invToReturn: rules.BlockDeviceInventory{
			Devices: []rules.BlockDevice{
				{Path: "/dev/sda1", DeviceID: "8:1", FSType: "vfat", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", UUID: "1111-2222"},
				{Path: "/dev/sda2", DeviceID: "8:2", FSType: "ext4", PartType: "0fc63daf-8483-4772-8e79-3d69d8477de4", UUID: "2222-3333"},
				{Path: "/dev/sda3", DeviceID: "8:3", FSType: "vfat", PartType: "ebd0a0a2-b9e5-4433-87c0-68b6b72699c7", Label: "datalabel"},
			},
			Available: true,
		},
	}

	entries := []*rules.FstabEntry{
		{
			RawLine: "UUID=1111-2222 /boot/efi vfat defaults 1 2",
			LineNum: 1,
			Fields:  []string{"UUID=1111-2222", "/boot/efi", "vfat", "defaults", "1", "2"},
		},
		{
			RawLine: "UUID=2222-3333 /boot ext4 defaults 1 2",
			LineNum: 2,
			Fields:  []string{"UUID=2222-3333", "/boot", "ext4", "defaults", "1", "2"},
		},
		{
			RawLine: "LABEL=datalabel /data vfat defaults 1 2",
			LineNum: 3,
			Fields:  []string{"LABEL=datalabel", "/data", "vfat", "defaults", "1", "2"},
		},
	}

	issues, err := lintEntries(entries, api)
	if err != nil {
		t.Fatalf("lintEntries failed: %v", err)
	}

	if api.getBlockDevicesCount != 1 {
		t.Fatalf("expected GetBlockDevices to be called exactly 1 time across all entries, got %d", api.getBlockDevicesCount)
	}

	// All entries should validate cleanly without issues
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for valid entries, got %v", issues)
	}
}

func TestLintEntries_MemoizationIntegration_NoRelevantEntries(t *testing.T) {
	api := &countingAPI{
		invToReturn: rules.BlockDeviceInventory{
			Available: true,
		},
	}

	entries := []*rules.FstabEntry{
		{
			RawLine: "tmpfs /tmp tmpfs defaults 0 0",
			LineNum: 1,
			Fields:  []string{"tmpfs", "/tmp", "tmpfs", "defaults", "0", "0"},
		},
		{
			RawLine: "proc /proc proc defaults 0 0",
			LineNum: 2,
			Fields:  []string{"proc", "/proc", "proc", "defaults", "0", "0"},
		},
		{
			RawLine: "/dev/mapper/vg-root / ext4 defaults 0 1",
			LineNum: 3,
			Fields:  []string{"/dev/mapper/vg-root", "/", "ext4", "defaults", "0", "1"},
		},
	}

	_, err := lintEntries(entries, api)
	if err != nil {
		t.Fatalf("lintEntries failed: %v", err)
	}

	if api.getBlockDevicesCount != 0 {
		t.Fatalf("expected GetBlockDevices to be called 0 times for non-local block entries, got %d", api.getBlockDevicesCount)
	}
}

func TestLintEntries_MemoizationIntegration_PreservesDiagnosticsAndErrors(t *testing.T) {
	api := &countingAPI{
		invToReturn: rules.BlockDeviceInventory{
			Available: false,
			Diagnostics: []rules.BlockDeviceDiagnostic{
				{Source: "/sys/class/block", Error: "permission denied"},
			},
		},
	}

	entries := []*rules.FstabEntry{
		{
			RawLine: "UUID=1111-2222 /boot/efi vfat defaults 1 2",
			LineNum: 1,
			Fields:  []string{"UUID=1111-2222", "/boot/efi", "vfat", "defaults", "1", "2"},
		},
	}

	issues, err := lintEntries(entries, api)
	if err != nil {
		t.Fatalf("lintEntries failed: %v", err)
	}

	if api.getBlockDevicesCount != 1 {
		t.Fatalf("expected GetBlockDevices to be called 1 time, got %d", api.getBlockDevicesCount)
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d: %v", len(issues), issues)
	}

	if issues[0].Level != rules.IssueLevelInfo {
		t.Errorf("expected IssueLevelInfo, got %v", issues[0].Level)
	}
	if issues[0].Message != "Live metadata check unavailable for this block device" {
		t.Errorf("unexpected issue message: %q", issues[0].Message)
	}
}

func TestLintEntries_MemoizationIntegration_InventoryErrorIsInformational(t *testing.T) {
	inventoryErr := errors.New("inventory provider failed")
	api := &countingAPI{errToReturn: inventoryErr}
	entries := []*rules.FstabEntry{
		{RawLine: "UUID=1111 /boot vfat defaults 0 2", Fields: []string{"UUID=1111", "/boot", "vfat", "defaults", "0", "2"}},
		{RawLine: "PARTLABEL=EFI\\040System /boot/efi vfat defaults 0 2", Fields: []string{"PARTLABEL=EFI\\040System", "/boot/efi", "vfat", "defaults", "0", "2"}},
	}

	issues, err := lintEntries(entries, api)
	if err != nil {
		t.Fatalf("lintEntries returned fatal error: %v", err)
	}
	if api.getBlockDevicesCount != 1 {
		t.Fatalf("GetBlockDevices called %d times, expected 1", api.getBlockDevicesCount)
	}
	if len(issues) != 2 {
		t.Fatalf("got %d issues, expected 2: %v", len(issues), issues)
	}
	for _, issue := range issues {
		if issue.Level != rules.IssueLevelInfo {
			t.Errorf("issue level = %v, expected info", issue.Level)
		}
		if !strings.Contains(issue.Message, inventoryErr.Error()) {
			t.Errorf("issue %q does not preserve provider error", issue.Message)
		}
	}
}
