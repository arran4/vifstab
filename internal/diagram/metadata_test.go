package diagram

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"math"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
)

type inventoryAPI struct {
	rules.API
	inventory rules.BlockDeviceInventory
	err       error
	calls     int
}

func (a *inventoryAPI) GetBlockDevices() (rules.BlockDeviceInventory, error) {
	a.calls++
	return a.inventory, a.err
}

type sourceOS struct {
	rules.OSProvider
	info fs.FileInfo
	err  error
}

func (o sourceOS) Stat(string) (fs.FileInfo, error) { return o.info, o.err }

func sourceInfo(t *testing.T, mode fs.FileMode) fs.FileInfo {
	t.Helper()
	info, err := fstest.MapFS{"source": {Mode: mode}}.Stat("source")
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func u64(value uint64) *uint64 { return &value }

func TestBlockDiagramNeverOpensDevice(t *testing.T) {
	for _, tc := range []struct {
		name      string
		inventory rules.BlockDeviceInventory
		err       error
	}{
		{"unavailable", rules.BlockDeviceInventory{}, os.ErrPermission},
		{"partial", rules.BlockDeviceInventory{Available: true, Devices: []rules.BlockDevice{{Path: "/dev/fixture", DeviceID: "8:0"}}}, nil},
		{"metadata", rules.BlockDeviceInventory{Available: true, Devices: []rules.BlockDevice{{Path: "/dev/fixture", DeviceID: "8:0", Layout: &rules.BlockLayout{SizeBytes: u64(51200)}}}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &inventoryAPI{inventory: tc.inventory, err: tc.err}
			deps := Dependencies{API: api, OS: sourceOS{info: sourceInfo(t, fs.ModeDevice)}, OpenImage: func(string, ...diskfs.OpenOpt) (*disk.Disk, error) {
				t.Fatal("block device reached image opener")
				return nil, nil
			}}
			var out bytes.Buffer
			if err := Print(&out, "/dev/fixture", deps, deps.OpenImage); err != nil {
				t.Fatal(err)
			}
			if api.calls != 1 || !strings.Contains(out.String(), "no block-device contents read") || !strings.Contains(out.String(), "Partition layout: unknown") || strings.Contains(out.String(), "0 partitions") || strings.Contains(out.String(), "[-----") {
				t.Fatalf("misleading metadata diagram: %s", out.String())
			}
		})
	}
}

func TestMetadataPartitionOrderHandlesUnknownNumbers(t *testing.T) {
	root := rules.BlockDevice{Path: "/dev/root", DeviceID: "8:0", Layout: &rules.BlockLayout{SizeBytes: u64(100)}}
	children := []rules.BlockDevice{
		{Path: "/dev/a", DeviceID: "8:2", Layout: &rules.BlockLayout{ParentDeviceID: "8:0", PartitionNumber: u64(2)}},
		{Path: "/dev/b", DeviceID: "8:3", Layout: &rules.BlockLayout{ParentDeviceID: "8:0"}},
		{Path: "/dev/c", DeviceID: "8:1", Layout: &rules.BlockLayout{ParentDeviceID: "8:0", PartitionNumber: u64(1)}},
	}
	var forward, reverse bytes.Buffer
	inv := rules.BlockDeviceInventory{Available: true, Devices: children}
	if err := printBlockMetadata(&forward, &root, inv, nil); err != nil {
		t.Fatal(err)
	}
	inv.Devices = []rules.BlockDevice{children[2], children[1], children[0]}
	if err := printBlockMetadata(&reverse, &root, inv, nil); err != nil {
		t.Fatal(err)
	}
	if forward.String() != reverse.String() || !strings.Contains(forward.String(), "A /dev/c") || !strings.Contains(forward.String(), "C /dev/b") {
		t.Fatalf("partial numbering made order nondeterministic: %s", forward.String())
	}
}

func TestBlockDiagramCanonicalIdentityLookup(t *testing.T) {
	api := &inventoryAPI{inventory: rules.BlockDeviceInventory{Available: true, Devices: []rules.BlockDevice{{Path: "/dev/fixture", UUID: "fixture-uuid", Aliases: []string{"/dev/disk/by-id/fixture"}, Layout: &rules.BlockLayout{SizeBytes: u64(0)}}}}}
	for _, ref := range []string{"UUID=fixture-uuid", "/dev/disk/by-id/fixture"} {
		var out bytes.Buffer
		err := Print(&out, ref, Dependencies{API: api, OS: sourceOS{err: os.ErrNotExist}, OpenImage: func(string, ...diskfs.OpenOpt) (*disk.Disk, error) {
			t.Fatal("metadata reference reached image opener")
			return nil, nil
		}})
		if err != nil || !strings.Contains(out.String(), "Device: /dev/fixture") || !strings.Contains(out.String(), "0 bytes") {
			t.Fatalf("canonical identity lost: %s %v", out.String(), err)
		}
	}
}

func TestDiagramClassificationUnknownAndNonregular(t *testing.T) {
	for _, mode := range []fs.FileMode{fs.ModeDir, fs.ModeNamedPipe, fs.ModeDevice | fs.ModeCharDevice} {
		api := &inventoryAPI{}
		if err := Print(io.Discard, "/fixture", Dependencies{API: api, OS: sourceOS{info: sourceInfo(t, mode)}, OpenImage: func(string, ...diskfs.OpenOpt) (*disk.Disk, error) {
			t.Fatal("nonregular input opened")
			return nil, nil
		}}); err == nil || api.calls != 0 {
			t.Fatalf("nonregular source accepted or probed: mode=%v err=%v", mode, err)
		}
	}
	for _, err := range []error{os.ErrPermission, errors.New("unavailable")} {
		api := &inventoryAPI{}
		got := Print(io.Discard, "/fixture", Dependencies{API: api, OS: sourceOS{err: err}, OpenImage: func(string, ...diskfs.OpenOpt) (*disk.Disk, error) { t.Fatal("unknown input opened"); return nil, nil }})
		if got == nil || !strings.Contains(got.Error(), "unknown") || !errors.Is(got, err) {
			t.Fatalf("unknown became absence/success: %v", got)
		}
	}
}

type failingDiagramWriter struct{ short bool }

func (w failingDiagramWriter) Write(p []byte) (int, error) {
	if w.short {
		return 0, nil
	}
	return 0, io.ErrClosedPipe
}

func TestMetadataDiagramWriterErrorsAndBounds(t *testing.T) {
	inv := rules.BlockDeviceInventory{Available: true, Devices: []rules.BlockDevice{
		{Path: "/dev/root", DeviceID: "8:0", Layout: &rules.BlockLayout{SizeBytes: u64(math.MaxUint64)}},
		{Path: "/dev/part", DeviceID: "8:1", Layout: &rules.BlockLayout{ParentDeviceID: "8:0", StartBytes: u64(math.MaxUint64 - 9), SizeBytes: u64(20), PartitionNumber: u64(1)}},
	}}
	var out bytes.Buffer
	if err := printBlockMetadata(&out, &inv.Devices[0], inv, nil); err != nil || !strings.Contains(out.String(), "Position unknown") {
		t.Fatalf("overflowing range not unknown: %s %v", out.String(), err)
	}
	inv.Devices[1].Layout.SizeBytes = u64(9)
	if scaledPosition(math.MaxUint64, math.MaxUint64) != 60 || scaledPosition(0, math.MaxUint64) != 0 {
		t.Fatal("scaled position overflow")
	}
	for _, short := range []bool{false, true} {
		err := printBlockMetadata(failingDiagramWriter{short: short}, &inv.Devices[0], inv, nil)
		want := io.ErrClosedPipe
		if short {
			want = io.ErrShortWrite
		}
		if !errors.Is(err, want) {
			t.Fatalf("writer failure lost: %v", err)
		}
	}
}
