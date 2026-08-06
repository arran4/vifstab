package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/partition/mbr"
	"github.com/arran4/vifstab/internal/txtar"
	"github.com/arran4/vifstab/internal/diagram"
)

func TestDiagram(t *testing.T) {
	testfiles, err := filepath.Glob("testdata/diagram/*.txtar")
	if err != nil {
		t.Fatal(err)
	}

	for _, testfile := range testfiles {
		t.Run(filepath.Base(testfile), func(t *testing.T) {
			a, err := txtar.ParseFile(testfile)
			if err != nil {
				t.Fatal(err)
			}

			// Define input variables
			var diskSize int64
			var tableType string
			var parts []string // format: "start,size"

			for _, file := range a.Files {
				switch file.Name {
				case "input/diskSize":
					s := strings.TrimSpace(string(file.Data))
					diskSize, _ = strconv.ParseInt(s, 10, 64)
				case "input/tableType":
					tableType = strings.TrimSpace(string(file.Data))
				case "input/partitions":
					lines := strings.Split(string(file.Data), "\n")
					for _, line := range lines {
						line = strings.TrimSpace(line)
						if line != "" {
							parts = append(parts, line)
						}
					}
				}
			}

			// Build actual mock file and partition it
			tmpFile, err := os.CreateTemp("", "diskfs-*.img")
			if err != nil {
				t.Fatal(err)
			}
			tmpPath := tmpFile.Name()
			_ = tmpFile.Truncate(diskSize)
			tmpFile.Close()
			defer os.Remove(tmpPath)

			d, err := diskfs.Open(tmpPath)
			if err != nil {
				t.Fatal(err)
			}

			if tableType == "gpt" {
				table := &gpt.Table{
					LogicalSectorSize:  512,
					PhysicalSectorSize: 512,
				}
				for _, p := range parts {
					ps := strings.Split(p, ",")
					start, _ := strconv.ParseUint(ps[0], 10, 64)
					size, _ := strconv.ParseUint(ps[1], 10, 64)
					table.Partitions = append(table.Partitions, &gpt.Partition{
						Start: start / 512,
						Size:  size,
						Type:  gpt.LinuxFilesystem,
						Name:  "part",
					})
				}
				d.Table = table
			} else if tableType == "mbr" {
				table := &mbr.Table{
					LogicalSectorSize:  512,
					PhysicalSectorSize: 512,
				}
				for _, p := range parts {
					ps := strings.Split(p, ",")
					start, _ := strconv.ParseUint(ps[0], 10, 64)
					size, _ := strconv.ParseUint(ps[1], 10, 64)
					table.Partitions = append(table.Partitions, &mbr.Partition{
						Start: uint32(start / 512),
						Size:  uint32(size / 512),
						Type:  mbr.Linux,
					})
				}
				err = d.Partition(table)
				if err != nil {
					t.Fatalf("failed to write mbr table: %v", err)
				}
			}

			// Mock disk opener that returns the mock image
			var mockOpener diagram.DiskFSCreator = func(device string, opts ...diskfs.OpenOpt) (*disk.Disk, error) {
				return d, nil
			}

			var buf bytes.Buffer
			err = diagram.Print(&buf, "mock_device", mockOpener)
			if err != nil {
				t.Fatal(err)
			}

			var expectedOutput string
			for _, file := range a.Files {
				if file.Name == "expected/output.txt" {
					expectedOutput = string(file.Data)
				}
			}

			if buf.String() != expectedOutput {
				t.Errorf("expected:\n%sgot:\n%s", expectedOutput, buf.String())
			}
		})
	}
}
