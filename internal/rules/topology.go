package rules

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// BlockLayout describes the kernel/udev view, not an inspected on-disk table.
// Nil numeric fields and empty Kind/ParentDeviceID/TableType mean unknown.
// These observations are currently excluded from capture schema v1.
type BlockLayout struct {
	Kind            string // "device", "partition", or unknown.
	SizeBytes       *uint64
	StartBytes      *uint64
	PartitionNumber *uint64
	ParentDeviceID  string
	TableType       string
}

func decimalMetadata(data []byte) (uint64, error) {
	text := strings.TrimSpace(string(data))
	if text == "" || strings.Trim(text, "0123456789") != "" {
		return 0, fmt.Errorf("invalid unsigned decimal metadata")
	}
	return strconv.ParseUint(text, 10, 64)
}

func validDeviceID(text string) bool {
	major, minor, ok := strings.Cut(text, ":")
	if !ok {
		return false
	}
	if major != strings.TrimSpace(major) || minor != strings.TrimSpace(minor) {
		return false
	}
	_, e1 := decimalMetadata([]byte(major))
	_, e2 := decimalMetadata([]byte(minor))
	return e1 == nil && e2 == nil && strings.TrimSpace(text) == text && !strings.ContainsAny(text, "/\\")
}

// enrichBlockLayout uses only the same injected metadata provider as inventory
// discovery. sysfs size/start are always counts of 512-byte sectors, regardless
// of the device's logical/physical sector size. No /dev object is opened.
func enrichBlockLayout(provider OSProvider, device *BlockDevice) []BlockDeviceDiagnostic {
	if device.Layout == nil {
		device.Layout = &BlockLayout{}
	}
	layout := device.Layout
	base := ""
	if name := device.SysfsName; name != "" && name != "." && name != ".." && filepath.Base(name) == name && !strings.ContainsRune(name, 0) {
		base = "/sys/class/block/" + name
	} else if validDeviceID(device.DeviceID) {
		base = "/sys/dev/block/" + device.DeviceID
	}
	if base == "" {
		return []BlockDeviceDiagnostic{{Source: device.Path, Error: "kernel topology identity unavailable; unknown"}}
	}
	var diagnostics []BlockDeviceDiagnostic
	unknown := func(path string, err error) {
		diagnostics = append(diagnostics, BlockDeviceDiagnostic{Source: path, Error: "topology unknown: " + err.Error()})
	}
	readSectors := func(name string) *uint64 {
		path := base + "/" + name
		data, err := provider.ReadFile(path)
		if err != nil {
			unknown(path, err)
			return nil
		}
		n, err := decimalMetadata(data)
		if err != nil || n > math.MaxUint64/512 {
			unknown(path, fmt.Errorf("invalid or overflowing sector count"))
			return nil
		}
		bytes := n * 512
		return &bytes
	}
	layout.SizeBytes = readSectors("size")
	partitionPath := base + "/partition"
	partitionData, err := provider.ReadFile(partitionPath)
	if err != nil {
		if os.IsNotExist(err) && layout.SizeBytes != nil {
			// Recheck that the sysfs object still exists: disappearing metadata
			// must not be mistaken for a definitely absent partition attribute.
			info, statErr := provider.Stat(base)
			if statErr == nil && info != nil && info.IsDir() {
				layout.Kind = "device"
			} else if statErr != nil {
				unknown(base, statErr)
			} else {
				unknown(base, fmt.Errorf("expected kernel metadata directory"))
			}
		} else {
			unknown(partitionPath, err)
		}
		return diagnostics
	}
	number, err := decimalMetadata(partitionData)
	if err != nil || number == 0 || number > math.MaxInt32 {
		unknown(partitionPath, fmt.Errorf("invalid partition number"))
		return diagnostics
	}
	layout.Kind, layout.PartitionNumber = "partition", &number
	layout.StartBytes = readSectors("start")
	target, err := provider.Readlink(base)
	if err != nil {
		unknown(base, err)
		return diagnostics
	}
	resolved := target
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(base), resolved)
	}
	resolved = filepath.Clean(resolved)
	if !strings.HasPrefix(resolved, "/sys/devices/") {
		unknown(base, fmt.Errorf("unexpected kernel topology link target"))
		return diagnostics
	}
	parentPath := filepath.Join(filepath.Dir(resolved), "dev")
	parentData, err := provider.ReadFile(parentPath)
	if err != nil {
		unknown(parentPath, err)
		return diagnostics
	}
	parentID := strings.TrimSpace(string(parentData))
	if !validDeviceID(parentID) || parentID == device.DeviceID {
		unknown(parentPath, fmt.Errorf("invalid or cyclic parent device identity"))
		return diagnostics
	}
	layout.ParentDeviceID = parentID
	return diagnostics
}
