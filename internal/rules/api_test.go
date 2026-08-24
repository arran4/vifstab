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
	cases := []struct {
		name          string
		fsys          fstest.MapFS
		symlinks      map[string]string
		checkInv      func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS)
	}{
		{
			name: "sysfs baseline discovery",
			fsys: fstest.MapFS{
				"sys/class/block/vda": &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
				"run/udev/data": &fstest.MapFile{Mode: fs.ModeDir}, // empty udev dir
			},
			symlinks: nil,
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if !inv.Available { t.Errorf("expected Available=true") }
				if len(inv.Devices) != 1 { t.Fatalf("expected 1 device, got %d", len(inv.Devices)) }
				if inv.Devices[0].Path != "/dev/vda" || inv.Devices[0].DeviceID != "254:0" {
					t.Errorf("expected basic sysfs device /dev/vda, 254:0")
				}
			},
		},
		{
			name: "two sysfs devices with only one usable udev record",
			fsys: fstest.MapFS{
				"sys/class/block/vda": &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
				"sys/class/block/vdb": &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vdb/dev": &fstest.MapFile{Data: []byte("254:16\n")},
				"run/udev/data/b254:0": &fstest.MapFile{
					Data: []byte("E:ID_FS_TYPE=ext4\n"),
				},
			},
			symlinks: map[string]string{
				"/sys/dev/block/254:0": "../../devices/virtual/block/vda",
			},
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if !inv.Available { t.Errorf("expected Available=true") }
				if len(inv.Devices) != 2 { t.Fatalf("expected 2 devices, got %d", len(inv.Devices)) }

				var vda, vdb *rules.BlockDevice
				for i := range inv.Devices {
					if inv.Devices[i].Path == "/dev/vda" { vda = &inv.Devices[i] }
					if inv.Devices[i].Path == "/dev/vdb" { vdb = &inv.Devices[i] }
				}

				if vda == nil || vdb == nil { t.Fatalf("missing vda or vdb") }
				if vda.FSType != "ext4" { t.Errorf("expected vda to be enriched to ext4") }
				if vdb.FSType != "" { t.Errorf("expected vdb to have no FSType") }
			},
		},
		{
			name: "UUID, LABEL, PARTUUID enrichment and resolution",
			fsys: fstest.MapFS{
				"sys/class/block/vda": &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
				"dev/disk/by-uuid/1111-2222": &fstest.MapFile{},
				"dev/disk/by-label/gentooboot": &fstest.MapFile{},
				"dev/disk/by-partuuid/aaaa-bbbb": &fstest.MapFile{},
			},
			symlinks: map[string]string{
				"/dev/disk/by-uuid/1111-2222": "../../vda",
				"/dev/disk/by-label/gentooboot": "../../vda",
				"/dev/disk/by-partuuid/aaaa-bbbb": "../../vda",
			},
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if !inv.Available { t.Errorf("expected Available=true") }
				if len(inv.Devices) != 1 { t.Fatalf("expected 1 device, got %d", len(inv.Devices)) }
				dev := inv.Devices[0]
				if dev.UUID != "1111-2222" { t.Errorf("expected UUID enrichment") }
				if dev.Label != "gentooboot" { t.Errorf("expected Label enrichment") }
				if dev.PartUUID != "aaaa-bbbb" { t.Errorf("expected PartUUID enrichment") }

				// Inventory resolution
				if inv.Resolve("UUID=1111-2222") == nil || inv.Resolve("UUID=1111-2222").Path != "/dev/vda" { t.Errorf("failed UUID resolution") }
				if inv.Resolve("LABEL=gentooboot") == nil || inv.Resolve("LABEL=gentooboot").Path != "/dev/vda" { t.Errorf("failed LABEL resolution") }
				if inv.Resolve("PARTUUID=aaaa-bbbb") == nil || inv.Resolve("PARTUUID=aaaa-bbbb").Path != "/dev/vda" { t.Errorf("failed PARTUUID resolution") }
				if inv.Resolve("/dev/disk/by-uuid/1111-2222") == nil || inv.Resolve("/dev/disk/by-uuid/1111-2222").Path != "/dev/vda" { t.Errorf("failed /dev/disk/by-uuid resolution") }
				if inv.Resolve("/dev/disk/by-label/gentooboot") == nil || inv.Resolve("/dev/disk/by-label/gentooboot").Path != "/dev/vda" { t.Errorf("failed /dev/disk/by-label resolution") }
				if inv.Resolve("/dev/disk/by-partuuid/aaaa-bbbb") == nil || inv.Resolve("/dev/disk/by-partuuid/aaaa-bbbb").Path != "/dev/vda" { t.Errorf("failed /dev/disk/by-partuuid resolution") }
				if inv.Resolve("/dev/vda") == nil || inv.Resolve("/dev/vda").Path != "/dev/vda" { t.Errorf("failed /dev/vda resolution") }
			},
		},
		{
			name: "mount source using UUID/by-* alias attaches to canonical device",
			fsys: fstest.MapFS{
				"sys/class/block/vda": &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
				"dev/disk/by-uuid/1111-2222": &fstest.MapFile{},
				"proc/mounts": &fstest.MapFile{
					Data: []byte("/dev/disk/by-uuid/1111-2222 /mnt ext4 rw 0 0\nUUID=1111-2222 /mnt2 ext4 rw 0 0\n"),
				},
			},
			symlinks: map[string]string{
				"/dev/disk/by-uuid/1111-2222": "../../vda",
			},
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if len(inv.Devices) != 1 { t.Fatalf("expected 1 device") }
				dev := inv.Devices[0]
				if len(dev.MountPoints) != 2 { t.Fatalf("expected 2 mountpoints on canonical device, got %v", dev.MountPoints) }
				if dev.MountPoints[0] != "/mnt" || dev.MountPoints[1] != "/mnt2" { t.Errorf("unexpected mountpoints: %v", dev.MountPoints) }
			},
		},
		{
			name: "sysfs unavailable + aliases readable + no usable udev -> Available=false",
			fsys: fstest.MapFS{
				"dev/disk/by-uuid/1111-2222": &fstest.MapFile{},
			},
			symlinks: map[string]string{
				"/dev/disk/by-uuid/1111-2222": "../../vda",
			},
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if inv.Available { t.Errorf("expected Available=false") }
				if len(inv.Devices) != 0 { t.Errorf("expected 0 devices") }
			},
		},
		{
			name: "inventory Resolve of unknown path never touches host filesystem",
			fsys: fstest.MapFS{
				"sys/class/block/vda": &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
			},
			symlinks: nil,
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if inv.Resolve("/dev/sdb") != nil { t.Errorf("expected unknown resolution to be nil") }
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := rules.ActualFS{
				OS: MockOSProvider{fsys: tc.fsys, symlinks: tc.symlinks},
			}
			inv, err := api.GetBlockDevices()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.checkInv(t, inv, api)
		})
	}
}
