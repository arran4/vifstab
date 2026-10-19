package txtar

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/rules"
)

type dummyOS struct{}

func (d dummyOS) ReadDir(name string) ([]os.DirEntry, error) { return nil, os.ErrNotExist }
func (d dummyOS) Open(name string) (os.File, error)          { return os.File{}, os.ErrNotExist }
func (d dummyOS) Readlink(name string) (string, error)       { return "", os.ErrNotExist }

type mockAPI struct{}

func (m mockAPI) CheckFileExists(path string) rules.FileExistsResult {
	if path == "/tmp/test" {
		return rules.FileExists
	}
	return rules.FileDoesNotExist
}

func (m mockAPI) CheckBlockDeviceExists(dev string) rules.ExistState {
	if dev == "/dev/sda1" {
		return rules.Exists
	}
	return rules.DoesNotExist
}

func (m mockAPI) GetHostKernelVersion() string {
	return "Linux version 5.15.0-100-generic"
}

func (m mockAPI) CheckUserExists(uid int) rules.ExistState  { return rules.Unknown }
func (m mockAPI) CheckGroupExists(gid int) rules.ExistState { return rules.Unknown }

func (m mockAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	return rules.BlockDeviceInventory{
		Devices: []rules.BlockDevice{
			{
				Path:    "/dev/sda1",
				UUID:    "1234-5678",
				Label:   "MY_LABEL",
				Aliases: []string{"/dev/disk/by-uuid/1234-5678", "/dev/disk/by-label/MY_LABEL"},
			},
		},
		Available: true,
	}, nil
}

func (m mockAPI) ResolveDevicePath(dev string) string {
	return dev
}

func TestCaptureAndReplay(t *testing.T) {
	tempDir := t.TempDir()
	fstabPath := filepath.Join(tempDir, "fstab")

	fstabContent := `/dev/sda1 /tmp/test ext4 defaults,unknown_opt 0 0`
	if err := os.WriteFile(fstabPath, []byte(fstabContent), 0644); err != nil {
		t.Fatal(err)
	}

	api := mockAPI{}

	// Test Capture
	data, err := CaptureSystem(context.Background(), api, fstabPath, false)
	if err != nil {
		t.Fatalf("CaptureSystem failed: %v", err)
	}

	// Test Parse
	snapshot, err := ParseTXTAR(data)
	if err != nil {
		t.Fatalf("ParseTXTAR failed: %v", err)
	}

	if strings.TrimSpace(string(snapshot.Fstab)) != fstabContent {
		t.Errorf("expected fstab %q, got %q", fstabContent, string(snapshot.Fstab))
	}

	if snapshot.KernelVer != "Linux version 5.15.0-100-generic" {
		t.Errorf("expected kernel %q, got %q", "Linux version 5.15.0-100-generic", snapshot.KernelVer)
	}

	// Check sanitized inside inventory
	invBytes := snapshot.MockFS["provider/inventory.json"].Data
	blockDevicesData := string(invBytes)
	if !strings.Contains(blockDevicesData, "sanitized") {
		t.Errorf("expected block devices to be sanitized, got %q", blockDevicesData)
	}

	// Test Replay
	replayAPI, err := ReplaySnapshot(snapshot)
	if err != nil {
		t.Fatalf("ReplaySnapshot failed: %v", err)
	}

	if replayAPI.GetHostKernelVersion() != "Linux version 5.15.0-100-generic" {
		t.Errorf("replay kernel version mismatch: %v", replayAPI.GetHostKernelVersion())
	}

	if replayAPI.CheckFileExists("/tmp/test") != rules.FileExists {
		t.Errorf("replay CheckFileExists returned incorrect result")
	}

	// Test offline isolation
	if replayAPI.CheckFileExists("/unmocked/path") != rules.Unknown {
		t.Errorf("expected unmocked path to return Unknown offline, got %v", replayAPI.CheckFileExists("/unmocked/path"))
	}
}

func TestParseTXTARLimits(t *testing.T) {
	// Size limit test
	largeData := make([]byte, 10*1024*1024+1)
	_, err := ParseTXTAR(largeData)
	if err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("expected size limit error, got %v", err)
	}

	// Unsafe path test
	unsafeData := []byte("-- input/../etc/fstab --\nhello")
	_, err = ParseTXTAR(unsafeData)
	if err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Errorf("expected unsafe path error, got %v", err)
	}
}
