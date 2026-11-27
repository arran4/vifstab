package rules_test

import (
	"encoding/json"
	"io/fs"
	"math"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules"
)

type topologyOS struct {
	MockOSProvider
	readErrors map[string]error
	statErrors map[string]error
	reads      []string
}

func (p *topologyOS) ReadFile(path string) ([]byte, error) {
	p.reads = append(p.reads, path)
	if strings.HasPrefix(path, "/dev/") {
		panic("device contents must never be read")
	}
	if err := p.readErrors[path]; err != nil {
		return nil, err
	}
	return p.MockOSProvider.ReadFile(path)
}

func (p *topologyOS) Open(path string) (fs.File, error) {
	if strings.HasPrefix(path, "/dev/") {
		panic("device must never be opened")
	}
	return p.MockOSProvider.Open(path)
}

func (p *topologyOS) Stat(path string) (fs.FileInfo, error) {
	if err := p.statErrors[path]; err != nil {
		return nil, err
	}
	return p.MockOSProvider.Stat(path)
}

func topologyProvider() *topologyOS {
	return &topologyOS{MockOSProvider: MockOSProvider{
		fsys: fstest.MapFS{
			"sys/class/block/vda/dev":           {Data: []byte("8:0\n")},
			"sys/class/block/vda/size":          {Data: []byte("100\n")},
			"sys/class/block/vda1/dev":          {Data: []byte("8:1\n")},
			"sys/class/block/vda1/size":         {Data: []byte("20\n")},
			"sys/class/block/vda1/start":        {Data: []byte("10\n")},
			"sys/class/block/vda1/partition":    {Data: []byte("1\n")},
			"sys/devices/virtual/block/vda/dev": {Data: []byte("8:0\n")},
			"run/udev/data/b8:0":                {Data: []byte("E:DEVNAME=/dev/vda\nE:ID_PART_TABLE_TYPE=gpt\n")},
			"run/udev/data/b8:1":                {Data: []byte("E:DEVNAME=/dev/vda1\nE:ID_FS_TYPE=ext4\n")},
			"dev/disk/by-uuid/fixture-disk":     {Mode: fs.ModeSymlink},
			"proc/mounts":                       {Data: nil},
		},
		symlinks: map[string]string{
			"/sys/class/block/vda1":          "../../devices/virtual/block/vda/vda1",
			"/dev/disk/by-uuid/fixture-disk": "../../vda",
		},
	}}
}

func TestBlockTopologyUsesCanonicalInventory(t *testing.T) {
	provider := topologyProvider()
	api := rules.ActualFS{OS: provider, IncludeTopology: true}
	inv, err := api.GetBlockDevices()
	if err != nil {
		t.Fatal(err)
	}
	root, child := inv.Resolve("UUID=fixture-disk"), inv.Resolve("/dev/vda1")
	if root == nil || child == nil || root.Layout == nil || child.Layout == nil {
		t.Fatalf("missing canonical objects: %+v", inv)
	}
	if root.Layout.Kind != "device" || root.Layout.SizeBytes == nil || *root.Layout.SizeBytes != 51200 || root.Layout.TableType != "gpt" {
		t.Fatalf("root layout: %+v", root.Layout)
	}
	if child.Layout.Kind != "partition" || child.Layout.SizeBytes == nil || *child.Layout.SizeBytes != 10240 || child.Layout.StartBytes == nil || *child.Layout.StartBytes != 5120 || child.Layout.PartitionNumber == nil || *child.Layout.PartitionNumber != 1 || child.Layout.ParentDeviceID != root.DeviceID {
		t.Fatalf("partition layout: %+v", child.Layout)
	}
	if len(inv.Diagnostics) != 0 {
		t.Fatalf("complete fixture produced unknown observations: %v", inv.Diagnostics)
	}
	data, err := json.Marshal(root)
	if err != nil || strings.Contains(string(data), "layout") || strings.Contains(string(data), "TableType") || strings.Contains(string(data), "51200") {
		t.Fatalf("live diagram topology must not silently extend capture schema v1: %s %v", data, err)
	}
}

func TestBlockTopologyIsExplicitEnrichment(t *testing.T) {
	provider := topologyProvider()
	inv, err := (rules.ActualFS{OS: provider}).GetBlockDevices()
	if err != nil {
		t.Fatal(err)
	}
	for _, device := range inv.Devices {
		if device.Layout != nil {
			t.Fatal("normal validation unexpectedly requested topology")
		}
	}
	for _, path := range provider.reads {
		if strings.HasSuffix(path, "/size") || strings.HasSuffix(path, "/start") || strings.HasSuffix(path, "/partition") {
			t.Fatalf("normal inventory read diagram-only metadata: %s", path)
		}
	}
}

func TestBlockTopologyUnknownAndMalformed(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(*topologyOS)
		check func(*testing.T, *rules.BlockDevice, *rules.BlockDevice)
	}{
		{"permission denied size", func(p *topologyOS) { p.readErrors = map[string]error{"/sys/class/block/vda/size": os.ErrPermission} }, func(t *testing.T, root, child *rules.BlockDevice) {
			if root.Layout.SizeBytes != nil || root.Layout.Kind != "" {
				t.Fatal("unreadable size invented a complete object")
			}
		}},
		{"malformed size", func(p *topologyOS) { p.fsys["sys/class/block/vda/size"].Data = []byte("10 junk\n") }, func(t *testing.T, root, child *rules.BlockDevice) {
			if root.Layout.SizeBytes != nil {
				t.Fatal("malformed size was accepted")
			}
		}},
		{"overflow sectors", func(p *topologyOS) { p.fsys["sys/class/block/vda/size"].Data = []byte("18446744073709551615\n") }, func(t *testing.T, root, child *rules.BlockDevice) {
			if root.Layout.SizeBytes != nil {
				t.Fatal("overflowing sectors accepted")
			}
		}},
		{"negative start", func(p *topologyOS) { p.fsys["sys/class/block/vda1/start"].Data = []byte("-1\n") }, func(t *testing.T, root, child *rules.BlockDevice) {
			if child.Layout.StartBytes != nil || child.Layout.ParentDeviceID != root.DeviceID {
				t.Fatal("invalid start lost other independent enrichment")
			}
		}},
		{"truncated partition", func(p *topologyOS) { p.fsys["sys/class/block/vda1/partition"].Data = nil }, func(t *testing.T, root, child *rules.BlockDevice) {
			if child.Layout.Kind != "" || child.Layout.PartitionNumber != nil {
				t.Fatal("truncated partition number became a device/partition")
			}
		}},
		{"out-of-domain partition number", func(p *topologyOS) { p.fsys["sys/class/block/vda1/partition"].Data = []byte("2147483648\n") }, func(t *testing.T, root, child *rules.BlockDevice) {
			if child.Layout.Kind != "" || child.Layout.PartitionNumber != nil {
				t.Fatal("partition number outside the kernel signed-int domain accepted")
			}
		}},
		{"partition permission denied", func(p *topologyOS) {
			p.readErrors = map[string]error{"/sys/class/block/vda/partition": os.ErrPermission}
		}, func(t *testing.T, root, child *rules.BlockDevice) {
			if root.Layout.Kind != "" {
				t.Fatal("inaccessible attribute became absent")
			}
		}},
		{"disappeared sysfs object", func(p *topologyOS) { p.statErrors = map[string]error{"/sys/class/block/vda": os.ErrNotExist} }, func(t *testing.T, root, child *rules.BlockDevice) {
			if root.Layout.Kind != "" {
				t.Fatal("disappeared sysfs object became a whole device")
			}
		}},
		{"cyclic parent", func(p *topologyOS) { p.fsys["sys/devices/virtual/block/vda/dev"].Data = []byte("8:1\n") }, func(t *testing.T, root, child *rules.BlockDevice) {
			if child.Layout.ParentDeviceID != "" {
				t.Fatal("cyclic parent accepted")
			}
		}},
		{"invalid parent identity", func(p *topologyOS) { p.fsys["sys/devices/virtual/block/vda/dev"].Data = []byte("8 :0\n") }, func(t *testing.T, root, child *rules.BlockDevice) {
			if child.Layout.ParentDeviceID != "" {
				t.Fatal("malformed parent accepted")
			}
		}},
		{"unexpected topology path", func(p *topologyOS) { p.symlinks["/sys/class/block/vda1"] = "/outside/devices/vda1" }, func(t *testing.T, root, child *rules.BlockDevice) {
			if child.Layout.ParentDeviceID != "" {
				t.Fatal("non-kernel parent path accepted")
			}
		}},
		{"unrecognized table label", func(p *topologyOS) {
			p.fsys["run/udev/data/b8:0"].Data = []byte("E:DEVNAME=/dev/vda\nE:ID_PART_TABLE_TYPE=unrecognized\n")
		}, func(t *testing.T, root, child *rules.BlockDevice) {
			if root.Layout.TableType != "" {
				t.Fatal("unknown table label became a known format")
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider := topologyProvider()
			tc.edit(provider)
			inv, err := (rules.ActualFS{OS: provider, IncludeTopology: true}).GetBlockDevices()
			if err != nil || len(inv.Diagnostics) == 0 {
				t.Fatalf("missing unknown diagnostics: %v %v", inv.Diagnostics, err)
			}
			root, child := inv.Resolve("/dev/vda"), inv.Resolve("/dev/vda1")
			if root == nil || child == nil {
				t.Fatal("partial metadata erased observed devices")
			}
			tc.check(t, root, child)
		})
	}
}

func TestBlockTopologyZeroAndLargestSectorCounts(t *testing.T) {
	for _, text := range []string{"0\n", "36028797018963967\n"} {
		provider := topologyProvider()
		provider.fsys["sys/class/block/vda/size"].Data = []byte(text)
		inv, err := (rules.ActualFS{OS: provider, IncludeTopology: true}).GetBlockDevices()
		root := inv.Resolve("/dev/vda")
		if err != nil || root == nil || root.Layout.SizeBytes == nil || *root.Layout.SizeBytes > math.MaxUint64-511 {
			t.Fatalf("valid sector boundary: %+v %v", root, err)
		}
	}
}

func TestBlockTopologyEnrichesUdevIdentityWithoutClassEnumeration(t *testing.T) {
	provider := topologyProvider()
	for path := range provider.fsys {
		if strings.HasPrefix(path, "sys/class/block/") {
			delete(provider.fsys, path)
		}
	}
	for path, value := range map[string]string{
		"sys/dev/block/8:0/size":      "100\n",
		"sys/dev/block/8:1/size":      "20\n",
		"sys/dev/block/8:1/start":     "10\n",
		"sys/dev/block/8:1/partition": "1\n",
	} {
		provider.fsys[path] = &fstest.MapFile{Data: []byte(value)}
	}
	provider.symlinks["/sys/dev/block/8:1"] = "../../devices/virtual/block/vda/vda1"
	inv, err := (rules.ActualFS{OS: provider, IncludeTopology: true}).GetBlockDevices()
	root, child := inv.Resolve("UUID=fixture-disk"), inv.Resolve("/dev/vda1")
	if err != nil || root == nil || child == nil || root.Layout.SizeBytes == nil || *root.Layout.SizeBytes != 51200 || child.Layout.ParentDeviceID != root.DeviceID || len(inv.Diagnostics) == 0 {
		t.Fatalf("independent sources were not preserved: %+v %v", inv, err)
	}
}

func TestBlockTopologyRejectsPartiallyReadTableMetadata(t *testing.T) {
	provider := topologyProvider()
	provider.fsys["run/udev/data/b8:0"].Data = []byte("E:DEVNAME=/dev/vda\nE:ID_PART_TABLE_TYPE=gpt\n" + strings.Repeat("x", 70*1024))
	inv, err := (rules.ActualFS{OS: provider, IncludeTopology: true}).GetBlockDevices()
	root := inv.Resolve("/dev/vda")
	if err != nil || root == nil || root.Layout.TableType != "" || root.Layout.SizeBytes == nil || len(inv.Diagnostics) == 0 {
		t.Fatalf("partial udev file became a known table or erased kernel observations: %+v %v", inv, err)
	}
}

var _ rules.OSProvider = (*topologyOS)(nil)
