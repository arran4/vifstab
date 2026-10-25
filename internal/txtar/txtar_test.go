package txtar

import (
	"context"
	"fmt"
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

func TestFullArchivePrivacy(t *testing.T) {
	tempDir := t.TempDir()
	fstabPath := filepath.Join(tempDir, "fstab")

	fstabContent := `
# server=TOP_SECRET_HOST password=TOP_SECRET_PASS
/dev/disk/by-id/wwn-0x50014ee058778899 /mnt/data ext4 defaults 0 2
UUID=SECRET-UUID /mnt/data2 ext4 defaults 0 2
//SECRET_NAS/share /mnt/net cifs credentials=/home/user/.smbcredentials,password=TOP_SECRET_PASS 0 0
/dev/mapper/PRIVATE_MAPPER /mnt/data3 ext4 defaults 0 2
`
	if err := os.WriteFile(fstabPath, []byte(fstabContent), 0644); err != nil {
		t.Fatal(err)
	}

	api := mockAPI{}
	data, err := CaptureSystem(context.Background(), api, fstabPath, false)
	if err != nil {
		t.Fatalf("CaptureSystem failed: %v", err)
	}

	outStr := string(data)
	if strings.Contains(outStr, "TOP_SECRET_HOST") {
		t.Errorf("expected TOP_SECRET_HOST to be redacted, got leaked in archive: %q", outStr)
	}
	if strings.Contains(outStr, "TOP_SECRET_PASS") {
		t.Errorf("expected TOP_SECRET_PASS to be redacted, got leaked in archive")
	}
	if strings.Contains(outStr, "wwn-0x50014ee058778899") {
		t.Errorf("expected wwn-0x50014ee058778899 to be redacted, got leaked in archive")
	}
	if strings.Contains(outStr, "SECRET-UUID") {
		t.Errorf("expected SECRET-UUID to be redacted, got leaked in archive")
	}
	if strings.Contains(outStr, "SECRET_NAS") {
		t.Errorf("expected SECRET_NAS to be redacted, got leaked in archive")
	}
	if strings.Contains(outStr, "PRIVATE_MAPPER") {
		t.Errorf("expected PRIVATE_MAPPER to be redacted, got leaked in archive")
	}

	// Verify Redacted Fstab
	snapshot, err := ParseTXTAR(data)
	if err != nil {
		t.Fatalf("ParseTXTAR failed: %v", err)
	}
	fstabOut := string(snapshot.Fstab)
	if !strings.Contains(fstabOut, "UUID=") {
		t.Errorf("expected fstab to retain UUID= prefix, got: %q", fstabOut)
	}
	if !strings.Contains(fstabOut, "//") {
		t.Errorf("expected fstab to retain CIFS // prefix, got: %q", fstabOut)
	}
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

	redactedTarget := ""
	lines := strings.Split(string(snapshot.Fstab), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if len(l) > 0 && !strings.HasPrefix(l, "#") && !strings.HasPrefix(l, "<sanitized-malformed") {
			parts := strings.Fields(l)
			if len(parts) >= 2 {
				redactedTarget = parts[1]
				break
			}
		}
	}

	if redactedTarget != "" {
		if replayAPI.CheckFileExists(redactedTarget) != rules.FileExists {
			t.Errorf("replay CheckFileExists returned incorrect result for sanitized target: %s", redactedTarget)
		}
	} else {
		t.Errorf("could not find sanitized path in fstab: %q", string(snapshot.Fstab))
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
		t.Fatalf("expected empty fstab to be valid, got %v", err)
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


// multiDeviceAPI supplies a fully controlled block-device inventory for
// redacted capture and replay tests (no live host device access).
type multiDeviceAPI struct {
	mockAPI
	inventory rules.BlockDeviceInventory
	err       error
}

func (m multiDeviceAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	return m.inventory, m.err
}

func (m multiDeviceAPI) CheckBlockDeviceExists(dev string) rules.ExistState {
	if m.inventory.Resolve(dev) != nil {
		return rules.Exists
	}
	return rules.Unknown
}

func TestCaptureRedactedAliasesResolveTwoDevices(t *testing.T) {
	inv := rules.BlockDeviceInventory{
		Available: true,
		Devices: []rules.BlockDevice{
			{
				Path: "/dev/mapper/PRIVATE_MAPPER_A", UUID: "PRIVATE_UUID_A",
				Label: "PRIVATE_LABEL_A", PartUUID: "PRIVATE_PARTUUID_A",
				Aliases: []string{
					"/dev/disk/by-uuid/PRIVATE_UUID_A",
					"/dev/disk/by-label/PRIVATE_LABEL_A",
					"/dev/disk/by-partuuid/PRIVATE_PARTUUID_A",
					"/dev/disk/by-partlabel/PRIVATE_PARTLABEL_A",
					"/dev/disk/by-id/PRIVATE_SERIAL_A",
					"/dev/disk/by-path/PRIVATE_TOPOLOGY_A",
				},
			},
			{
				Path: "/dev/sdb2", UUID: "PRIVATE_UUID_B",
				Label: "PRIVATE_LABEL_B", PartUUID: "PRIVATE_PARTUUID_B",
				Aliases: []string{
					"/dev/disk/by-uuid/PRIVATE_UUID_B",
					"/dev/disk/by-label/PRIVATE_LABEL_B",
					"/dev/disk/by-partuuid/PRIVATE_PARTUUID_B",
					"/dev/disk/by-partlabel/PRIVATE_PARTLABEL_B",
					"/dev/disk/by-id/PRIVATE_SERIAL_B",
					"/dev/disk/by-path/PRIVATE_TOPOLOGY_B",
				},
			},
		},
	}
	cases := []struct {
		source string
		device int
	}{
		{"/dev/mapper/PRIVATE_MAPPER_A", 0},
		{"UUID=PRIVATE_UUID_A", 0},
		{"LABEL=PRIVATE_LABEL_B", 1},
		{"PARTUUID=PRIVATE_PARTUUID_B", 1},
		{"PARTLABEL=PRIVATE_PARTLABEL_A", 0},
		{"ID=PRIVATE_SERIAL_B", 1},
		{"/dev/disk/by-uuid/PRIVATE_UUID_B", 1},
		{"/dev/disk/by-label/PRIVATE_LABEL_A", 0},
		{"/dev/disk/by-partuuid/PRIVATE_PARTUUID_A", 0},
		{"/dev/disk/by-partlabel/PRIVATE_PARTLABEL_B", 1},
		{"/dev/disk/by-id/PRIVATE_SERIAL_A", 0},
		{"/dev/disk/by-path/PRIVATE_TOPOLOGY_B", 1},
		{"/dev/sdb2", 1},
	}
	var lines []string
	for i, tc := range cases {
		opts := "defaults"
		if i == 0 {
			opts = "defaults,credentials=/home/PRIVATE_USER/PRIVATE_CREDENTIAL_FILE,password=PRIVATE_PASS_A"
		}
		lines = append(lines, fmt.Sprintf("%s /mnt/PRIVATE_MOUNT_%d ext4 %s 0 2", tc.source, i, opts))
	}
	fstabPath := filepath.Join(t.TempDir(), "fstab")
	if err := os.WriteFile(fstabPath, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	raw, err := CaptureSystem(context.Background(), multiDeviceAPI{inventory: inv}, fstabPath, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{
		"PRIVATE_MAPPER_A", "PRIVATE_UUID_A", "PRIVATE_UUID_B",
		"PRIVATE_LABEL_A", "PRIVATE_LABEL_B", "PRIVATE_PARTUUID_A",
		"PRIVATE_PARTUUID_B", "PRIVATE_PARTLABEL_A", "PRIVATE_PARTLABEL_B",
		"PRIVATE_SERIAL_A", "PRIVATE_SERIAL_B", "PRIVATE_TOPOLOGY_A",
		"PRIVATE_TOPOLOGY_B", "PRIVATE_USER", "PRIVATE_CREDENTIAL_FILE",
		"PRIVATE_PASS_A", "PRIVATE_MOUNT_", "/dev/sdb2",
	} {
		if strings.Contains(string(raw), secret) {
			t.Errorf("redacted archive leaks %q", secret)
		}
	}
	snapshot, err := ParseTXTAR(raw)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := ReplaySnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	replayedInv, err := replay.GetBlockDevices()
	if err != nil || len(replayedInv.Devices) != 2 {
		t.Fatalf("replayed inventory missing two devices: %#v, %v", replayedInv, err)
	}
	redactedLines := strings.Split(strings.TrimSpace(string(snapshot.Fstab)), "\n")
	if len(redactedLines) != len(cases) {
		t.Fatalf("got %d fstab entries, want %d: %q", len(redactedLines), len(cases), snapshot.Fstab)
	}
	for i, line := range redactedLines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			t.Fatalf("invalid redacted fstab entry: %q", line)
		}
		dev := replayedInv.Resolve(fields[0])
		if dev == nil || dev.Path != replayedInv.Devices[cases[i].device].Path {
			t.Errorf("line %d: %q resolves to %#v, want device %d", i, fields[0], dev, cases[i].device)
		}
		if state := replay.CheckBlockDeviceExists(fields[0]); state != rules.Exists {
			t.Errorf("line %d: block existence = %v, want Exists", i, state)
		}
	}
	if state := replay.CheckBlockDeviceExists("/dev/NOT_IN_ARCHIVE"); state != rules.Unknown {
		t.Errorf("unknown device should remain Unknown offline, got %v", state)
	}
}

func TestCaptureRedactsFailedInventoryDiagnostics(t *testing.T) {
	fstabPath := filepath.Join(t.TempDir(), "fstab")
	if err := os.WriteFile(fstabPath, []byte("PARTLABEL=PRIVATE_PARTLABEL_FAIL /mnt/PRIVATE_TARGET ext4 defaults 0 2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	api := multiDeviceAPI{err: fmt.Errorf("PRIVATE_PROVIDER_DIAGNOSTIC")}
	data, err := CaptureSystem(context.Background(), api, fstabPath, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"PRIVATE_PROVIDER_DIAGNOSTIC", "PRIVATE_PARTLABEL_FAIL", "PRIVATE_TARGET"} {
		if strings.Contains(string(data), secret) {
			t.Errorf("failed inventory capture leaked %q", secret)
		}
	}
}

func TestReplayKeepsBlockAndFileExistenceIndependent(t *testing.T) {
	for _, tc := range []struct {
		name string
		file string
		block string
		wantFile rules.FileExistsResult
		wantBlock rules.ExistState
	}{
		{"device-exists-file-missing", "false", "true", rules.FileDoesNotExist, rules.Exists},
		{"file-exists-device-missing", "true", "false", rules.FileExists, rules.DoesNotExist},
		{"file-exists-device-unknown", "true", "unknown", rules.FileExists, rules.Unknown},
		{"file-missing-device-permission", "false", "permission_denied", rules.FileDoesNotExist, rules.PermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := &Snapshot{
				Fstab: []byte("/dev/PRIVATE_DEVICE /mnt/test ext4 defaults 0 2\n"),
				Cache: map[string]string{
					"file_exists:/dev/PRIVATE_DEVICE": tc.file,
					"block_exists:/dev/PRIVATE_DEVICE": tc.block,
				},
			}
			api, err := ReplaySnapshot(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			if got := api.CheckFileExists("/dev/PRIVATE_DEVICE"); got != tc.wantFile {
				t.Errorf("file existence = %v, want %v", got, tc.wantFile)
			}
			if got := api.CheckBlockDeviceExists("/dev/PRIVATE_DEVICE"); got != tc.wantBlock {
				t.Errorf("block existence = %v, want %v", got, tc.wantBlock)
			}
		})
	}
}
