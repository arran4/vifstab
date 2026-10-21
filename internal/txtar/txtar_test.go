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

	fstabContent := `# server=private-host user=alice secret=TOP_SECRET

malformed-only
/dev/sda1 /tmp/test ext4 defaults,unknown_opt,password=TOP_SECRET 0 0`
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

	// Since we sanitize fstab properly now (by word boundary token replacement on unredacted=false),
	// it should be different from original fstab!
	if strings.TrimSpace(string(snapshot.Fstab)) == fstabContent {
		t.Errorf("expected fstab to be sanitized and differ from original %q, got %q", fstabContent, string(snapshot.Fstab))
	}
	if strings.Contains(string(snapshot.Fstab), "TOP_SECRET") {
		t.Errorf("fstab leaked secret TOP_SECRET: %q", string(snapshot.Fstab))
	}
	if strings.Contains(string(snapshot.Fstab), "private-host") {
		t.Errorf("fstab leaked private-host: %q", string(snapshot.Fstab))
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

	// The original path "/tmp/test" was renamed to something like "/sanitized-path-N"
	// So asking CheckFileExists for original path in replay should return Unknown or DoesNotExist,
	// while the *sanitized* path should return FileExists.
	// Since we don't know the exact N, we can just extract it from the fstab output.
	redactedTarget := ""
	lines := strings.Split(string(snapshot.Fstab), "\n")
	for _, l := range lines {
		if strings.Contains(l, "/sanitized-path-") {
			parts := strings.Fields(l)
			if len(parts) >= 2 {
				redactedTarget = parts[1]
			}
			break
		}
	}

	if redactedTarget != "" {
		if replayAPI.CheckFileExists(redactedTarget) != rules.FileExists {
			t.Errorf("replay CheckFileExists returned incorrect result for sanitized target: %s", redactedTarget)
		}
	} else {
		t.Errorf("could not find sanitized path in fstab")
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
	// Duplicate file test
	dupData := []byte("-- input/etc/fstab --\nhello\n-- input/etc/fstab --\nworld\n")
	_, err = ParseTXTAR(dupData)
	if err == nil || !strings.Contains(err.Error(), "duplicate file") {
		t.Errorf("expected duplicate file error, got %v", err)
	}

	// Missing fstab test
	missingData := []byte("-- kernel --\n123\n")
	_, err = ParseTXTAR(missingData)
	if err == nil || !strings.Contains(err.Error(), "missing fstab data") {
		t.Errorf("expected missing fstab error, got %v", err)
	}

	// Empty fstab test (should be valid)
	emptyFstabData := []byte("-- manifest.json --\n{\"version\": \"1.0\", \"type\": \"system\", \"redacted\": false}\n-- input/etc/fstab --\n-- kernel --\n123\n")
	snapshot2, err := ParseTXTAR(emptyFstabData)
	if err != nil {
		t.Errorf("expected empty fstab to be valid, got %v", err)
	}

	if snapshot2.Fstab == nil {
		t.Errorf("expected fstab to be present but empty")
	}

	missingRedacted := []byte("-- manifest.json --\n{\"version\": \"1.0\", \"type\": \"system\"}\n-- input/etc/fstab --\n")
	_, err = ParseTXTAR(missingRedacted)
	if err == nil || !strings.Contains(err.Error(), "missing required field") {
	    t.Errorf("expected missing redacted field error, got %v", err)
	}

	malformedManifest := []byte("-- manifest.json --\n{\"version: 1.0\"}\n-- input/etc/fstab --\n")
	_, err = ParseTXTAR(malformedManifest)
	if err == nil {
		t.Errorf("expected malformed manifest error")
	}

	invalidVersion := []byte("-- manifest.json --\n{\"version\": \"2.0\"}\n-- input/etc/fstab --\n")
	_, err = ParseTXTAR(invalidVersion)
	if err == nil {
		t.Errorf("expected unsupported manifest version error")
	}
}
