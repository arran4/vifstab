package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/editor"
	"github.com/arran4/vifstab/internal/outcome"
	"github.com/arran4/vifstab/internal/rules"
	"github.com/cpuguy83/go-md2man/v2/md2man"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
)

//go:embed manpage.md
var manPageMarkdown []byte

func main() {
	if len(os.Args) > 1 && os.Args[1] == "diagram" {
		diagramCmd := flag.NewFlagSet("diagram", flag.ExitOnError)
		diagramCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: %s diagram <device>\n", os.Args[0])
			diagramCmd.PrintDefaults()
		}
		diagramCmd.Parse(os.Args[2:])

		if diagramCmd.NArg() != 1 {
			diagramCmd.Usage()
			os.Exit(1)
		}

		device := diagramCmd.Arg(0)
		err := printDiagram(os.Stdout, device)
		if err != nil {
			handleError(err)
		}
		return
	}

	var (
		backupDir       string
		file            string
		generateManpage bool
		verifyOnly      bool
	)

	flag.StringVar(&backupDir, "backup-dir", "", "Directory to store RCS backups (defaults to a sensible location based on the file being edited and the user running the command)")
	flag.StringVar(&file, "file", "/etc/fstab", "File to edit")
	flag.BoolVar(&generateManpage, "generate-manpage", false, "Generate man page for vifstab to vifstab.1")
	flag.BoolVar(&verifyOnly, "verify", false, "Verify the file without opening an editor")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unexpected arguments: %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}

	if generateManpage {
		manPageTroff := md2man.Render(manPageMarkdown)
		if err := os.WriteFile("vifstab.1", manPageTroff, 0644); err != nil {
			handleError(err)
		}
		return
	}

	if backupDir == "" {
		if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
			backupDir = xdgCache + "/vifstab"
		} else if home, err := os.UserHomeDir(); err == nil {
			backupDir = home + "/.cache/vifstab"
		} else {
			backupDir = "/tmp/.vifstab"
		}
	}

	cfg := config.Config{
		BackupDir: backupDir,
		FilePath:  file,
	}

	if verifyOnly {
		issues, err := editor.VerifyAndLint(cfg, rules.ActualFS{})
		if err != nil {
			handleError(err)
		}

		var issueCount int
		for _, s := range issues {
			if s.Level == rules.IssueLevelError || s.Level == rules.IssueLevelWarning {
				issueCount++
			}
		}

		if issueCount == 0 {
			fmt.Println("Validation successful.")
			if len(issues) > 0 {
				fmt.Println("Issues:")
				for _, s := range issues {
					fmt.Printf(" - [%s] %s (Line: %s)\n", s.Level, s.Message, s.Line)
				}
			}
			return
		}

		fmt.Println("Validation issues found:")
		for _, s := range issues {
			fmt.Printf(" - [%s] %s (Line: %s)\n", s.Level, s.Message, s.Line)
		}

		os.Exit(1)
	}

	err := editor.EditAndLint(cfg, rules.ActualFS{})
	if err != nil {
		handleError(err)
	}
}

// DiskFSCreator represents a function that can open a disk given a device string and options.
// This allows us to inject a mock implementation for testing.
type DiskFSCreator func(device string, opts ...diskfs.OpenOpt) (*disk.Disk, error)

func printDiagram(w io.Writer, device string, opts ...any) error {
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

	fmt.Fprintf(w, "Partition Table: %s\n", table.Type())
	fmt.Fprintf(w, "Disk Size: %d bytes\n\n", diskSize)

	partitions := table.GetPartitions()

	barWidth := 60
	chars := make([]rune, barWidth)
	for i := range chars {
		chars[i] = '-' // free space
	}

	for i, p := range partitions {
		if p.GetSize() == 0 {
			continue
		}

		fmt.Fprintf(w, "Partition %d: Start %d, Size %d\n", i+1, p.GetStart(), p.GetSize())

		startRatio := float64(p.GetStart()) / float64(diskSize)
		sizeRatio := float64(p.GetSize()) / float64(diskSize)

		startPos := int(startRatio * float64(barWidth))
		sizeChars := int(sizeRatio * float64(barWidth))
		if sizeChars < 1 && p.GetSize() > 0 {
			sizeChars = 1
		}

		for i := startPos; i < startPos+sizeChars && i < barWidth; i++ {
			chars[i] = '#'
		}
	}

	fmt.Fprintf(w, "\nVisual Diagram:\n[%s]\n", string(chars))

	return nil
}

func handleError(err error) {
	var appOutcome *outcome.ApplicationOutcome
	if errors.As(err, &appOutcome) {
		if appOutcome.Err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", appOutcome.Err)
		}
		os.Exit(appOutcome.ExitCode)
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
