package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/arran4/vifstab/internal/diagram"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/partition/mbr"
	"golang.org/x/tools/txtar"
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
			if err := tmpFile.Truncate(diskSize); err != nil {
				t.Fatalf("failed to truncate temp file: %v", err)
			}
			if err := tmpFile.Close(); err != nil {
				t.Fatalf("failed to close temp file: %v", err)
			}
			defer func() { _ = os.Remove(tmpPath) }()

			d, err := diskfs.Open(tmpPath)
			if err != nil {
				t.Fatal(err)
			}

			switch tableType {
			case "gpt":
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
			case "mbr":
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

// We also import os/exec for these tests. If it was already imported, it's fine,
// but we just define buildBinary here. Since imports can't be appended easily,
// we'll just write it and use goimports.
func buildBinary(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "vifstab")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build: %v\n%s", err, out)
	}
	return binPath
}

func TestVerifyFallbackError(t *testing.T) {
	binPath := buildBinary(t)

	tempDir := t.TempDir()
	fstabPath := filepath.Join(tempDir, "fstab")
	err := os.WriteFile(fstabPath, []byte(""), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binPath, "--verify", "--file", fstabPath)
	cmd.Env = []string{
		"XDG_CACHE_HOME=",
		"HOME=",
		"USERPROFILE=",
		"PATH=" + os.Getenv("PATH"),
	}

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("verify failed unexpectedly: %v\nStderr: %s", err, stderr.String())
	}
}

func TestExplicitBackupDir(t *testing.T) {
	binPath := buildBinary(t)

	tempDir := t.TempDir()
	fstabPath := filepath.Join(tempDir, "fstab")
	backupDir := filepath.Join(tempDir, "backup")
	err := os.WriteFile(fstabPath, []byte(""), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binPath, "--verify", "--file", fstabPath, "--backup-dir", backupDir)
	cmd.Env = []string{
		"XDG_CACHE_HOME=",
		"HOME=",
		"USERPROFILE=",
		"PATH=" + os.Getenv("PATH"),
	}

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("verify with explicit backup dir failed: %v\nStderr: %s", err, stderr.String())
	}
}

func TestFormatInPlaceNoCache(t *testing.T) {
	binPath := buildBinary(t)

	tempDir := t.TempDir()
	fstabPath := filepath.Join(tempDir, "fstab")
	backupDir := filepath.Join(tempDir, "backup")
	err := os.WriteFile(fstabPath, []byte(""), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binPath, "format", "-i", "--file", fstabPath)
	cmd.Env = []string{
		"XDG_CACHE_HOME=",
		"HOME=",
		"USERPROFILE=",
		"PATH=" + os.Getenv("PATH"),
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err == nil {
		t.Fatalf("format -i without cache dir should have failed, but it succeeded")
	}
	if !strings.Contains(stderr.String(), "could not determine user cache directory") {
		t.Fatalf("expected error about cache dir, got: %s", stderr.String())
	}

	cmd2 := exec.Command(binPath, "format", "-i", "--file", fstabPath, "--backup-dir", backupDir)
	cmd2.Env = []string{
		"XDG_CACHE_HOME=",
		"HOME=",
		"USERPROFILE=",
		"PATH=" + os.Getenv("PATH"),
	}
	var stderr2 bytes.Buffer
	cmd2.Stderr = &stderr2
	err = cmd2.Run()
	if err != nil {
		t.Fatalf("format -i with explicit backup dir failed: %v\nStderr: %s", err, stderr2.String())
	}
}
