package diagram

import (
	"fmt"
	"io"
	"io/fs"
	"reflect"
	"text/template"

	"github.com/arran4/vifstab/internal/rules"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

// DiskFSCreator opens a regular disk image after source classification.
// This allows us to inject a mock implementation for testing.
type DiskFSCreator func(device string, opts ...diskfs.OpenOpt) (*disk.Disk, error)

// Dependencies keeps host metadata and regular-image I/O injectable. Device
// discovery and identity resolution always use the canonical rules inventory.
type Dependencies struct {
	API       rules.API
	OS        rules.OSProvider
	OpenImage DiskFSCreator
}

const diagramTemplate = `Partition Table: {{.TableType}}
Disk Size: {{.DiskSize}} bytes
{{range .Partitions}}
Partition {{.Index}}: Start {{.Start}}, Size {{.Size}}{{if .FSType}}
  Type: {{.FSType}}{{end}}{{end}}

Visual Diagram:
[{{.VisualDiagram}}]
`

// Print outputs an ASCII diagram of the disk's partition table.
func Print(w io.Writer, device string, opts ...any) error {
	if device == "" {
		return fmt.Errorf("diagram requires an image path or block-device reference")
	}
	deps := Dependencies{}

	// Check if a mock diskfs creator was provided
	for _, opt := range opts {
		if d, ok := opt.(Dependencies); ok {
			deps = d
		}
		if creator, ok := opt.(DiskFSCreator); ok {
			deps.OpenImage = creator
		}
	}
	if deps.OS == nil {
		deps.OS = (rules.ActualFS{}).Provider()
	}
	if deps.API == nil {
		deps.API = rules.ActualFS{OS: deps.OS, IncludeTopology: true}
	}
	if deps.OpenImage == nil {
		deps.OpenImage = openRegularImage
	}
	info, statErr := deps.OS.Stat(device)
	if statErr == nil && nilPointer(info) {
		statErr = fmt.Errorf("metadata provider returned no file information")
	}
	if statErr != nil || info.Mode()&fs.ModeDevice != 0 && info.Mode()&fs.ModeCharDevice == 0 {
		inventory, inventoryErr := deps.API.GetBlockDevices()
		resolved := inventory.Resolve(device)
		if resolved != nil || statErr == nil {
			if resolved == nil {
				resolved = &rules.BlockDevice{Path: device}
			}
			return printBlockMetadata(w, resolved, inventory, inventoryErr)
		}
		return fmt.Errorf("input classification/identity unknown: %w", statErr)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("input must be a regular disk image or block device")
	}

	d, err := deps.OpenImage(device, diskfs.WithOpenMode(diskfs.ReadOnly))
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("image opener returned no disk")
	}
	if d.Backend != nil {
		defer func() { _ = d.Close() }()
	}

	table := d.Table
	if table == nil {
		table, err = d.GetPartitionTable()
		if err != nil {
			return err
		}
	}
	if nilPointer(table) || d.Size <= 0 {
		return fmt.Errorf("image partition table or size unavailable")
	}

	diskSize := d.Size

	type PartitionData struct {
		Index  int
		Start  int64
		Size   int64
		FSType string
	}

	type DiagramData struct {
		TableType     string
		DiskSize      int64
		Partitions    []PartitionData
		VisualDiagram string
	}

	data := DiagramData{
		TableType: table.Type(),
		DiskSize:  diskSize,
	}

	partitions := table.GetPartitions()

	barWidth := 60
	chars := make([]rune, barWidth)
	for i := range chars {
		chars[i] = '-' // free space
	}

	lastOccupiedPos := -1

	for i, p := range partitions {
		if nilPointer(p) || p.GetStart() < 0 || p.GetSize() < 0 {
			return fmt.Errorf("invalid image partition bounds at index %d", i+1)
		}
		if p.GetStart() > diskSize || p.GetSize() > diskSize-p.GetStart() {
			return fmt.Errorf("image partition outside image bounds at index %d", i+1)
		}
		if p.GetSize() == 0 {
			continue
		}

		var fsType string
		switch pt := p.(type) {
		case *gpt.Partition:
			fsType = string(pt.Type)
		case *mbr.Partition:
			fsType = fmt.Sprintf("0x%02x", pt.Type)
		}

		data.Partitions = append(data.Partitions, PartitionData{
			Index:  i + 1,
			Start:  p.GetStart(),
			Size:   p.GetSize(),
			FSType: fsType,
		})

		startRatio := float64(p.GetStart()) / float64(diskSize)
		sizeRatio := float64(p.GetSize()) / float64(diskSize)

		startPos := int(startRatio * float64(barWidth))
		sizeChars := int(sizeRatio * float64(barWidth))
		if sizeChars < 1 && p.GetSize() > 0 {
			sizeChars = 1
		}

		if startPos <= lastOccupiedPos {
			startPos = lastOccupiedPos + 1
		}

		if startPos >= barWidth {
			startPos = barWidth - 1 // clamp to end
		}
		if startPos+sizeChars > barWidth {
			sizeChars = barWidth - startPos
		}

		var letter rune
		if i < 26 {
			letter = rune('A' + i)
		} else if i < 52 {
			letter = rune('a' + (i - 26))
		} else if i < 62 {
			letter = rune('0' + (i - 52))
		} else {
			letter = '#' // fallback
		}

		for j := startPos; j < startPos+sizeChars && j < barWidth; j++ {
			chars[j] = letter
			lastOccupiedPos = j
		}
	}

	data.VisualDiagram = string(chars)

	tmpl, err := template.New("diagram").Parse(diagramTemplate)
	if err != nil {
		return err
	}

	return tmpl.Execute(completeWriter{w}, data)
}

func nilPointer(value any) bool {
	return value == nil || reflect.ValueOf(value).Kind() == reflect.Pointer && reflect.ValueOf(value).IsNil()
}

type completeWriter struct{ io.Writer }

func (w completeWriter) Write(data []byte) (int, error) {
	n, err := w.Writer.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return n, err
}
