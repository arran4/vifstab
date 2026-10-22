package diagram

import (
	"fmt"
	"io"
	"text/template"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

// DiskFSCreator represents a function that can open a disk given a device string and options.
// This allows us to inject a mock implementation for testing.
type DiskFSCreator func(device string, opts ...diskfs.OpenOpt) (*disk.Disk, error)

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
	var openDisk DiskFSCreator = diskfs.Open

	// Check if a mock diskfs creator was provided
	for _, opt := range opts {
		if creator, ok := opt.(DiskFSCreator); ok {
			openDisk = creator
		}
	}

	d, err := openDisk(device, diskfs.WithOpenMode(diskfs.ReadOnly))
	if err != nil {
		return err
	}

	table := d.Table
	if table == nil {
		table, err = d.GetPartitionTable()
		if err != nil {
			return err
		}
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

	return tmpl.Execute(w, data)
}
