package rules_test

import (
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules"
)

type MockOSProvider struct {
	fsys     fstest.MapFS
	symlinks map[string]string
}

func (m MockOSProvider) ReadDir(name string) ([]fs.DirEntry, error) {
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	return m.fsys.ReadDir(name)
}

func (m MockOSProvider) Open(name string) (fs.File, error) {
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	return m.fsys.Open(name)
}

func (m MockOSProvider) ReadFile(name string) ([]byte, error) {
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	return m.fsys.ReadFile(name)
}

func (m MockOSProvider) Readlink(name string) (string, error) {
	if target, ok := m.symlinks[name]; ok {
		return target, nil
	}
	return "", os.ErrNotExist
}

func (m MockOSProvider) Stat(name string) (fs.FileInfo, error) {
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	return m.fsys.Stat(name)
}

func TestSyntheticProvider(t *testing.T) {
	fsys := fstest.MapFS{
		"run/udev/data/b254:0": &fstest.MapFile{
			Data: []byte("E:ID_PART_ENTRY_TYPE=c12a7328-f81f-11d2-ba4b-00a0c93ec93b\nE:ID_FS_TYPE=vfat\n"),
		},
		"sys/class/block/vda": &fstest.MapFile{Mode: fs.ModeDir},
		"sys/class/block/vda/dev": &fstest.MapFile{
			Data: []byte("254:0\n"),
		},
		"dev/disk/by-uuid/1111-2222": &fstest.MapFile{},
		"dev/disk/by-label/gentooboot": &fstest.MapFile{},
		"proc/mounts": &fstest.MapFile{
			Data: []byte("/dev/vda /boot/efi vfat rw 0 0\n"),
		},
	}
	symlinks := map[string]string{
		"/sys/dev/block/254:0":           "../../devices/platform/block/vda",
		"/dev/disk/by-uuid/1111-2222":    "../../vda",
		"/dev/disk/by-label/gentooboot": "../../vda",
	}

	api := rules.ActualFS{
		OS: MockOSProvider{fsys: fsys, symlinks: symlinks},
	}

	inv, err := api.GetBlockDevices()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(inv.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(inv.Devices))
	}

	dev := inv.Devices[0]
	if dev.Path != "/dev/vda" {
		t.Errorf("expected path /dev/vda, got %s", dev.Path)
	}
	if dev.DeviceID != "254:0" {
		t.Errorf("expected DeviceID 254:0, got %s", dev.DeviceID)
	}
	if dev.FSType != "vfat" {
		t.Errorf("expected FSType vfat, got %s", dev.FSType)
	}
	if dev.PartType != "c12a7328-f81f-11d2-ba4b-00a0c93ec93b" {
		t.Errorf("expected PartType ESP, got %s", dev.PartType)
	}
	if dev.UUID != "1111-2222" {
		t.Errorf("expected UUID 1111-2222, got %s", dev.UUID)
	}
	if dev.Label != "gentooboot" {
		t.Errorf("expected Label gentooboot, got %s", dev.Label)
	}
	if len(dev.MountPoints) != 1 || dev.MountPoints[0] != "/boot/efi" {
		t.Errorf("expected mountpoint /boot/efi, got %v", dev.MountPoints)
	}

	// Test partial failure: udev unavailable
	fsysPartial := fstest.MapFS{
		"sys/class/block/vda": &fstest.MapFile{Mode: fs.ModeDir},
		"sys/class/block/vda/dev": &fstest.MapFile{
			Data: []byte("254:0\n"),
		},
	}
	apiPartial := rules.ActualFS{OS: MockOSProvider{fsys: fsysPartial, symlinks: nil}}
	invPartial, err := apiPartial.GetBlockDevices()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invPartial.Devices) != 1 {
		t.Fatalf("expected 1 device from partial, got %d", len(invPartial.Devices))
	}
	if len(invPartial.Diagnostics) == 0 {
		t.Errorf("expected diagnostics for missing udev, got none")
	}

	devPartial := invPartial.Devices[0]
	if devPartial.Path != "/dev/vda" || devPartial.DeviceID != "254:0" {
		t.Errorf("expected basic sysfs device info")
	}

	// Test ResolveDevicePath canonical alias equality
	if resolved := api.ResolveDevicePath("UUID=1111-2222"); resolved != "/dev/vda" {
		t.Errorf("expected UUID resolution to /dev/vda, got %s", resolved)
	}
	if resolved := api.ResolveDevicePath("LABEL=gentooboot"); resolved != "/dev/vda" {
		t.Errorf("expected LABEL resolution to /dev/vda, got %s", resolved)
	}
	if resolved := api.ResolveDevicePath("/dev/disk/by-uuid/1111-2222"); resolved != "/dev/vda" {
		t.Errorf("expected by-uuid resolution to /dev/vda, got %s", resolved)
	}
}
