package diagram

import (
	"fmt"
	"io"
	"math/bits"
	"sort"
	"strings"

	"github.com/arran4/vifstab/internal/rules"
)

func metadataNumber(value *uint64) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%d bytes", *value)
}

func partitionLetter(index int) rune {
	switch {
	case index < 26:
		return 'A' + rune(index)
	case index < 52:
		return 'a' + rune(index-26)
	case index < 62:
		return '0' + rune(index-52)
	default:
		return '#'
	}
}

// scaledPosition cannot overflow even for the largest kernel byte counts.
// Callers establish offset <= total and total > 0 first.
func scaledPosition(offset, total uint64) int {
	hi, lo := bits.Mul64(offset, 60)
	q, _ := bits.Div64(hi, lo, total)
	return int(q)
}

func printBlockMetadata(w io.Writer, device *rules.BlockDevice, inventory rules.BlockDeviceInventory, inventoryErr error) error {
	var out strings.Builder
	fmt.Fprintln(&out, "Source: kernel/udev metadata; no block-device contents read")
	fmt.Fprintf(&out, "Device: %s\n", device.Path)
	layout := device.Layout
	if layout == nil {
		layout = &rules.BlockLayout{}
	}
	table := layout.TableType
	if table == "" {
		table = "unknown"
	}
	fmt.Fprintf(&out, "Partition table type (udev metadata): %s\n", table)
	fmt.Fprintf(&out, "Size (kernel metadata): %s\n", metadataNumber(layout.SizeBytes))
	if layout.Kind == "partition" {
		fmt.Fprintf(&out, "Selected object: partition; parent device identity: %s\n", unknownString(layout.ParentDeviceID))
		fmt.Fprintf(&out, "Start in parent (kernel metadata): %s\n", metadataNumber(layout.StartBytes))
	}
	var children []rules.BlockDevice
	for _, candidate := range inventory.Devices {
		if device.DeviceID != "" && candidate.DeviceID != device.DeviceID && candidate.Layout != nil && candidate.Layout.ParentDeviceID == device.DeviceID {
			children = append(children, candidate)
		}
	}
	sort.Slice(children, func(i, j int) bool {
		a, b := children[i].Layout, children[j].Layout
		if (a.PartitionNumber == nil) != (b.PartitionNumber == nil) {
			return a.PartitionNumber != nil
		}
		if a.PartitionNumber != nil && b.PartitionNumber != nil && *a.PartitionNumber != *b.PartitionNumber {
			return *a.PartitionNumber < *b.PartitionNumber
		}
		return children[i].Path < children[j].Path
	})
	bar := []rune(strings.Repeat("?", 60))
	if len(children) == 0 {
		fmt.Fprintln(&out, "Partition layout: unknown; no child ranges correlated in available metadata")
	} else {
		fmt.Fprintln(&out, "Observed child partitions (may be incomplete):")
	}
	for i, child := range children {
		part := child.Layout
		letter := partitionLetter(i)
		fmt.Fprintf(&out, "%c %s: Start %s, Size %s\n", letter, child.Path, metadataNumber(part.StartBytes), metadataNumber(part.SizeBytes))
		if child.PartType != "" {
			fmt.Fprintf(&out, "  Partition type (udev metadata): %s\n", child.PartType)
		}
		if child.FSType != "" {
			fmt.Fprintf(&out, "  Filesystem type (metadata): %s\n", child.FSType)
		}
		if layout.SizeBytes == nil || part.StartBytes == nil || part.SizeBytes == nil {
			continue
		}
		total, start, size := *layout.SizeBytes, *part.StartBytes, *part.SizeBytes
		if total == 0 {
			fmt.Fprintln(&out, "  Position unknown: observed parent size is zero")
			continue
		}
		if start > total || size > total-start {
			fmt.Fprintln(&out, "  Position unknown: range inconsistent with the observed parent size")
			continue
		}
		if size == 0 {
			continue
		}
		first, last := scaledPosition(start, total), scaledPosition(start+size, total)
		if first == 60 {
			continue
		}
		if last <= first {
			last = first + 1
		}
		for j := first; j < last; j++ {
			if bar[j] == '?' {
				bar[j] = letter
			} else if bar[j] != letter {
				bar[j] = '*'
			}
		}
	}
	fmt.Fprintln(&out, "Approximate kernel-view positions (? unknown; * multiple observations per cell):")
	fmt.Fprintf(&out, "[%s]\n", string(bar))
	fmt.Fprintln(&out, "Unmarked space is unknown, not proof of free space or a complete on-disk table.")
	diagnostics := append([]rules.BlockDeviceDiagnostic(nil), inventory.Diagnostics...)
	if inventoryErr != nil {
		diagnostics = append(diagnostics, rules.BlockDeviceDiagnostic{Source: "inventory", Error: inventoryErr.Error()})
	}
	if !inventory.Available {
		fmt.Fprintln(&out, "Inventory availability: unknown/unavailable")
	}
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Source != diagnostics[j].Source {
			return diagnostics[i].Source < diagnostics[j].Source
		}
		return diagnostics[i].Error < diagnostics[j].Error
	})
	for _, diagnostic := range diagnostics {
		fmt.Fprintf(&out, "Metadata diagnostic: %s: %s\n", diagnostic.Source, diagnostic.Error)
	}
	_, err := io.WriteString(completeWriter{w}, out.String())
	return err
}

func unknownString(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
