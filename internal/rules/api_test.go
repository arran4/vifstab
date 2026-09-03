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
		name     string
		fsys     fstest.MapFS
		symlinks map[string]string
		checkInv func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS)
	}{
		{
			name: "sysfs baseline discovery",
			fsys: fstest.MapFS{
				"sys/class/block/vda":     &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
				"run/udev/data":           &fstest.MapFile{Mode: fs.ModeDir}, // empty udev dir
			},
			symlinks: nil,
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if !inv.Available {
					t.Errorf("expected Available=true")
				}
				if len(inv.Devices) != 1 {
					t.Fatalf("expected 1 device, got %d", len(inv.Devices))
				}
				if inv.Devices[0].Path != "/dev/vda" || inv.Devices[0].DeviceID != "254:0" {
					t.Errorf("expected basic sysfs device /dev/vda, 254:0")
				}
			},
		},
		{
			name: "two sysfs devices with only one usable udev record",
			fsys: fstest.MapFS{
				"sys/class/block/vda":     &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
				"sys/class/block/vdb":     &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vdb/dev": &fstest.MapFile{Data: []byte("254:16\n")},
				"run/udev/data/b254:0": &fstest.MapFile{
					Data: []byte("E:ID_FS_TYPE=ext4\n"),
				},
			},
			symlinks: map[string]string{
				"/sys/dev/block/254:0": "../../devices/virtual/block/vda",
			},
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if !inv.Available {
					t.Errorf("expected Available=true")
				}
				if len(inv.Devices) != 2 {
					t.Fatalf("expected 2 devices, got %d", len(inv.Devices))
				}

				var vda, vdb *rules.BlockDevice
				for i := range inv.Devices {
					if inv.Devices[i].Path == "/dev/vda" {
						vda = &inv.Devices[i]
					}
					if inv.Devices[i].Path == "/dev/vdb" {
						vdb = &inv.Devices[i]
					}
				}

				if vda == nil || vdb == nil {
					t.Fatalf("missing vda or vdb")
				}
				if vda.FSType != "ext4" {
					t.Errorf("expected vda to be enriched to ext4")
				}
				if vdb.FSType != "" {
					t.Errorf("expected vdb to have no FSType")
				}
			},
		},
		{
			name: "multiple metadata paths for one DeviceID result in one logical device",
			fsys: fstest.MapFS{
				"sys/class/block/vda":     &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
				"run/udev/data/b254:0": &fstest.MapFile{
					Data: []byte("E:DEVNAME=/dev/vda\nE:ID_FS_TYPE=ext4\n"),
				},
			},
			symlinks: map[string]string{
				"/sys/dev/block/254:0": "../../devices/virtual/block/vda",
			},
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if len(inv.Devices) != 1 {
					t.Fatalf("expected 1 logical device, got %d", len(inv.Devices))
				}
				if inv.Devices[0].DeviceID != "254:0" || inv.Devices[0].Path != "/dev/vda" {
					t.Errorf("unexpected device: %+v", inv.Devices[0])
				}
			},
		},
		{
			name: "two different DeviceIDs result in two devices",
			fsys: fstest.MapFS{
				"sys/class/block/vda":     &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
				"sys/class/block/vdb":     &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vdb/dev": &fstest.MapFile{Data: []byte("254:16\n")},
			},
			symlinks: nil,
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if len(inv.Devices) != 2 {
					t.Fatalf("expected 2 devices, got %d", len(inv.Devices))
				}
			},
		},
		{
			name: "UUID, LABEL, PARTUUID enrichment and resolution",
			fsys: fstest.MapFS{
				"sys/class/block/vda":            &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev":        &fstest.MapFile{Data: []byte("254:0\n")},
				"dev/disk/by-uuid/1111-2222":     &fstest.MapFile{},
				"dev/disk/by-label/gentooboot":   &fstest.MapFile{},
				"dev/disk/by-partuuid/aaaa-bbbb": &fstest.MapFile{},
			},
			symlinks: map[string]string{
				"/dev/disk/by-uuid/1111-2222":     "../../vda",
				"/dev/disk/by-label/gentooboot":   "../../vda",
				"/dev/disk/by-partuuid/aaaa-bbbb": "../../vda",
			},
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if !inv.Available {
					t.Errorf("expected Available=true")
				}
				if len(inv.Devices) != 1 {
					t.Fatalf("expected 1 device, got %d", len(inv.Devices))
				}
				dev := inv.Devices[0]
				if dev.UUID != "1111-2222" {
					t.Errorf("expected UUID enrichment")
				}
				if dev.Label != "gentooboot" {
					t.Errorf("expected Label enrichment")
				}
				if dev.PartUUID != "aaaa-bbbb" {
					t.Errorf("expected PartUUID enrichment")
				}

				// Inventory resolution
				if inv.Resolve("UUID=1111-2222") == nil || inv.Resolve("UUID=1111-2222").Path != "/dev/vda" {
					t.Errorf("failed UUID resolution")
				}
				if inv.Resolve("UUID=\"1111-2222\"") == nil || inv.Resolve("UUID=\"1111-2222\"").Path != "/dev/vda" {
					t.Errorf("failed quoted UUID resolution")
				}
				if inv.Resolve("LABEL=gentooboot") == nil || inv.Resolve("LABEL=gentooboot").Path != "/dev/vda" {
					t.Errorf("failed LABEL resolution")
				}
				if inv.Resolve("PARTUUID=aaaa-bbbb") == nil || inv.Resolve("PARTUUID=aaaa-bbbb").Path != "/dev/vda" {
					t.Errorf("failed PARTUUID resolution")
				}
				if inv.Resolve("/dev/disk/by-uuid/1111-2222") == nil || inv.Resolve("/dev/disk/by-uuid/1111-2222").Path != "/dev/vda" {
					t.Errorf("failed /dev/disk/by-uuid resolution")
				}
				if inv.Resolve("/dev/disk/by-label/gentooboot") == nil || inv.Resolve("/dev/disk/by-label/gentooboot").Path != "/dev/vda" {
					t.Errorf("failed /dev/disk/by-label resolution")
				}
				if inv.Resolve("/dev/disk/by-partuuid/aaaa-bbbb") == nil || inv.Resolve("/dev/disk/by-partuuid/aaaa-bbbb").Path != "/dev/vda" {
					t.Errorf("failed /dev/disk/by-partuuid resolution")
				}
				if inv.Resolve("/dev/vda") == nil || inv.Resolve("/dev/vda").Path != "/dev/vda" {
					t.Errorf("failed /dev/vda resolution")
				}
			},
		},
		{
			name: "PARTLABEL and by-partlabel, ID and by-id, by-path",
			fsys: fstest.MapFS{
				"sys/class/block/vda":                                  &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev":                              &fstest.MapFile{Data: []byte("254:0\n")},
				"dev/disk/by-partlabel/EFI\\x20System\\x20Partition":   &fstest.MapFile{},
				"dev/disk/by-id/ata-VBOX_HARDDISK_VB0e7f7881-80718d05": &fstest.MapFile{},
				"dev/disk/by-path/pci-0000:00:1f.2-ata-1":              &fstest.MapFile{},
			},
			symlinks: map[string]string{
				"/dev/disk/by-partlabel/EFI\\x20System\\x20Partition":   "../../vda",
				"/dev/disk/by-id/ata-VBOX_HARDDISK_VB0e7f7881-80718d05": "../../vda",
				"/dev/disk/by-path/pci-0000:00:1f.2-ata-1":              "../../vda",
			},
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if len(inv.Devices) != 1 {
					t.Fatalf("expected 1 device")
				}
				if inv.Resolve("PARTLABEL=EFI\\040System\\040Partition") == nil || inv.Resolve("PARTLABEL=EFI\\040System\\040Partition").Path != "/dev/vda" {
					t.Errorf("failed PARTLABEL resolution")
				}
				if inv.Resolve("ID=ata-VBOX_HARDDISK_VB0e7f7881-80718d05") == nil || inv.Resolve("ID=ata-VBOX_HARDDISK_VB0e7f7881-80718d05").Path != "/dev/vda" {
					t.Errorf("failed ID resolution")
				}
				if inv.Resolve("/dev/disk/by-path/pci-0000:00:1f.2-ata-1") == nil || inv.Resolve("/dev/disk/by-path/pci-0000:00:1f.2-ata-1").Path != "/dev/vda" {
					t.Errorf("failed by-path resolution")
				}
				if inv.Resolve("/dev/disk/by-partlabel/EFI\\040System\\040Partition") == nil || inv.Resolve("/dev/disk/by-partlabel/EFI\\040System\\040Partition").Path != "/dev/vda" {
					t.Errorf("failed by-partlabel resolution")
				}
				if inv.Resolve("/dev/disk/by-id/ata-VBOX_HARDDISK_VB0e7f7881-80718d05") == nil || inv.Resolve("/dev/disk/by-id/ata-VBOX_HARDDISK_VB0e7f7881-80718d05").Path != "/dev/vda" {
					t.Errorf("failed by-id resolution")
				}
			},
		},
		{
			name: "exact alias matching",
			fsys: fstest.MapFS{
				"sys/class/block/vda":                     &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev":                 &fstest.MapFile{Data: []byte("254:0\n")},
				"dev/disk/by-id/ata-DISK_123":             &fstest.MapFile{},
				"dev/disk/by-partlabel/my-boot-partition": &fstest.MapFile{},
			},
			symlinks: map[string]string{
				"/dev/disk/by-id/ata-DISK_123":             "../../vda",
				"/dev/disk/by-partlabel/my-boot-partition": "../../vda",
			},
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				// Exact matching: ID=DISK_123 should NOT match ata-DISK_123
				if inv.Resolve("ID=DISK_123") != nil {
					t.Errorf("ID=DISK_123 should not have matched ata-DISK_123")
				}
				if inv.Resolve("ID=ata-DISK_123") == nil {
					t.Errorf("ID=ata-DISK_123 should match exactly")
				}

				// PARTLABEL=boot-partition should NOT match my-boot-partition
				if inv.Resolve("PARTLABEL=boot-partition") != nil {
					t.Errorf("PARTLABEL=boot-partition should not have matched my-boot-partition")
				}
				if inv.Resolve("PARTLABEL=my-boot-partition") == nil {
					t.Errorf("PARTLABEL=my-boot-partition should match exactly")
				}
			},
		},
		{
			name: "mount source using UUID/by-* alias attaches to canonical device",
			fsys: fstest.MapFS{
				"sys/class/block/vda":        &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev":    &fstest.MapFile{Data: []byte("254:0\n")},
				"dev/disk/by-uuid/1111-2222": &fstest.MapFile{},
				"proc/mounts": &fstest.MapFile{
					Data: []byte("/dev/disk/by-uuid/1111-2222 /mnt ext4 rw 0 0\nUUID=1111-2222 /mnt2 ext4 rw 0 0\n"),
				},
			},
			symlinks: map[string]string{
				"/dev/disk/by-uuid/1111-2222": "../../vda",
			},
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if len(inv.Devices) != 1 {
					t.Fatalf("expected 1 device")
				}
				dev := inv.Devices[0]
				if len(dev.MountPoints) != 2 {
					t.Fatalf("expected 2 mountpoints on canonical device, got %v", dev.MountPoints)
				}
				if dev.MountPoints[0] != "/mnt" || dev.MountPoints[1] != "/mnt2" {
					t.Errorf("unexpected mountpoints: %v", dev.MountPoints)
				}
			},
		},
		{
			name: "duplicate mountpoints deduplicate",
			fsys: fstest.MapFS{
				"sys/class/block/vda":     &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
				"proc/mounts": &fstest.MapFile{
					Data: []byte("/dev/vda /mnt ext4 rw 0 0\n/dev/vda /mnt ext4 rw 0 0\n"),
				},
			},
			symlinks: nil,
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if len(inv.Devices) != 1 {
					t.Fatalf("expected 1 device")
				}
				if len(inv.Devices[0].MountPoints) != 1 || inv.Devices[0].MountPoints[0] != "/mnt" {
					t.Errorf("expected deduplicated mountpoint, got %v", inv.Devices[0].MountPoints)
				}
			},
		},
		{
			name: "deterministic ordering regardless of discovery order",
			fsys: fstest.MapFS{
				"sys/class/block/vdz":     &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vdz/dev": &fstest.MapFile{Data: []byte("254:32\n")},
				"sys/class/block/vda":     &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
			},
			symlinks: nil,
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if len(inv.Devices) != 2 {
					t.Fatalf("expected 2 devices")
				}
				if inv.Devices[0].Path != "/dev/vda" || inv.Devices[1].Path != "/dev/vdz" {
					t.Errorf("expected sorted deterministic order /dev/vda, /dev/vdz; got %s, %s", inv.Devices[0].Path, inv.Devices[1].Path)
				}
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
				if inv.Available {
					t.Errorf("expected Available=false")
				}
				if len(inv.Devices) != 0 {
					t.Errorf("expected 0 devices")
				}
			},
		},
		{
			name: "inventory Resolve of unknown path never touches host filesystem",
			fsys: fstest.MapFS{
				"sys/class/block/vda":     &fstest.MapFile{Mode: fs.ModeDir},
				"sys/class/block/vda/dev": &fstest.MapFile{Data: []byte("254:0\n")},
			},
			symlinks: nil,
			checkInv: func(t *testing.T, inv rules.BlockDeviceInventory, api rules.ActualFS) {
				if inv.Resolve("/dev/sdb") != nil {
					t.Errorf("expected unknown resolution to be nil")
				}
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

func TestCacheFS_GetBlockDevices(t *testing.T) {
	t.Run("explicit empty cache is available", func(t *testing.T) {
		cache := rules.NewCache()
		cache.Set(rules.CacheKey("block_devices"), []rules.BlockDevice{})
		cfs := rules.NewCacheFS(cache, fstest.MapFS{})

		inv, err := cfs.GetBlockDevices()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !inv.Available {
			t.Errorf("expected Available=true for explicit cached empty devices")
		}
		if len(inv.Devices) != 0 {
			t.Errorf("expected 0 devices, got %d", len(inv.Devices))
		}
	})

	t.Run("missing cache is unavailable", func(t *testing.T) {
		cache := rules.NewCache()
		cfs := rules.NewCacheFS(cache, fstest.MapFS{})

		inv, err := cfs.GetBlockDevices()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inv.Available {
			t.Errorf("expected Available=false for missing cache")
		}
	})
}

// Add ActualFS tests overriding the getOS function
type mockErrorOS struct {
	statFunc     func(name string) (fs.FileInfo, error)
	readFileFunc func(name string) ([]byte, error)
	openFunc     func(name string) (fs.File, error)
}

func (m mockErrorOS) Stat(name string) (fs.FileInfo, error) {
	if m.statFunc != nil {
		return m.statFunc(name)
	}
	return nil, os.ErrNotExist
}

func (m mockErrorOS) ReadFile(name string) ([]byte, error) {
	if m.readFileFunc != nil {
		return m.readFileFunc(name)
	}
	return nil, os.ErrNotExist
}

func (m mockErrorOS) Open(name string) (fs.File, error) {
	if m.openFunc != nil {
		return m.openFunc(name)
	}
	return nil, os.ErrNotExist
}

func (m mockErrorOS) ReadDir(name string) ([]fs.DirEntry, error) {
	return nil, os.ErrNotExist
}

func (m mockErrorOS) Readlink(name string) (string, error) {
	return "", os.ErrNotExist
}

func TestActualFS_CheckFileExists(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected rules.ExistState
	}{
		{"Exists", nil, rules.FileExists},
		{"DoesNotExist", os.ErrNotExist, rules.FileDoesNotExist},
		{"PermissionDenied", os.ErrPermission, rules.FilePermissionDenied},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := rules.ActualFS{
				OS: mockErrorOS{
					statFunc: func(name string) (fs.FileInfo, error) {
						return nil, tc.err
					},
				},
			}
			result := a.CheckFileExists("somefile")
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestActualFS_CheckUserExists(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected rules.ExistState
	}{
		{"Passwd missing", os.ErrNotExist, rules.Unknown},
		{"PermissionDenied", os.ErrPermission, rules.PermissionDenied},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := rules.ActualFS{
				OS: mockErrorOS{
					openFunc: func(name string) (fs.File, error) {
						return nil, tc.err
					},
				},
			}
			result := a.CheckUserExists(1000)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestActualFS_CheckGroupExists(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected rules.ExistState
	}{
		{"Group missing", os.ErrNotExist, rules.Unknown},
		{"PermissionDenied", os.ErrPermission, rules.PermissionDenied},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := rules.ActualFS{
				OS: mockErrorOS{
					openFunc: func(name string) (fs.File, error) {
						return nil, tc.err
					},
				},
			}
			result := a.CheckGroupExists(1000)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestCacheFS_PreservesExistState(t *testing.T) {
	cache := rules.NewCache()
	// Set explicit ExistState values, overriding boolean collapse
	cache.Set("file_exists:restricted", rules.PermissionDenied)
	cache.Set("user_exists:999", rules.PermissionDenied)
	cache.Set("group_exists:999", rules.Unknown)

	c := rules.NewCacheFS(cache, fstest.MapFS{}) // mapFS shouldn't be consulted for cached values

	if st := c.CheckFileExists("restricted"); st != rules.PermissionDenied {
		t.Errorf("Expected cache to preserve PermissionDenied for file, got %v", st)
	}

	if st := c.CheckUserExists(999); st != rules.PermissionDenied {
		t.Errorf("Expected cache to preserve PermissionDenied for user, got %v", st)
	}

	if st := c.CheckGroupExists(999); st != rules.Unknown {
		t.Errorf("Expected cache to preserve Unknown for group, got %v", st)
	}
}
