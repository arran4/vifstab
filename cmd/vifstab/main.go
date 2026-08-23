package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/diagram"
	"github.com/arran4/vifstab/internal/editor"
	"github.com/arran4/vifstab/internal/format"
	"github.com/arran4/vifstab/internal/outcome"
	"github.com/arran4/vifstab/internal/rules"
	"github.com/cpuguy83/go-md2man/v2/md2man"
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
		_ = diagramCmd.Parse(os.Args[2:])

		if diagramCmd.NArg() != 1 {
			diagramCmd.Usage()
			os.Exit(1)
		}

		device := diagramCmd.Arg(0)
		err := diagram.Print(os.Stdout, device)
		if err != nil {
			handleError(err)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "format" {
		formatCmd := flag.NewFlagSet("format", flag.ExitOnError)
		fileFlag := formatCmd.String("file", "/etc/fstab", "File to format")
		inPlaceFlag := formatCmd.Bool("i", false, "Write formatted output back to the file (in-place)")
		formatCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: %s format [options]\n", os.Args[0])
			formatCmd.PrintDefaults()
		}
		_ = formatCmd.Parse(os.Args[2:])

		if formatCmd.NArg() > 0 {
			fmt.Fprintf(os.Stderr, "Error: unexpected arguments: %v\n", formatCmd.Args())
			formatCmd.Usage()
			os.Exit(1)
		}

		var in *os.File
		var err error

		// If formatting standard input
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			in = os.Stdin
			if *inPlaceFlag {
				handleError(fmt.Errorf("cannot use -i when reading from standard input"))
			}
		} else {
			in, err = os.Open(*fileFlag)
			if err != nil {
				handleError(fmt.Errorf("failed to open file %s: %w", *fileFlag, err))
			}
			defer func() { _ = in.Close() }()
		}

		if *inPlaceFlag {
			// Read the entire file content, format it, then overwrite
			tempFile, err := os.CreateTemp("", "vifstab-format-*")
			if err != nil {
				handleError(fmt.Errorf("failed to create temp file: %w", err))
			}
			defer func() { _ = os.Remove(tempFile.Name()) }()
			defer func() { _ = tempFile.Close() }()

			err = format.Format(in, tempFile)
			if err != nil {
				handleError(fmt.Errorf("format failed: %w", err))
			}

			// Reset the temporary file's offset so we can read from it
			if _, err := tempFile.Seek(0, 0); err != nil {
				handleError(fmt.Errorf("failed to seek temp file: %w", err))
			}

			content, err := io.ReadAll(tempFile)
			if err != nil {
				handleError(fmt.Errorf("failed to read formatted content: %w", err))
			}

			cfg := config.GetDefaultConfig(*fileFlag)
			if err := editor.SaveAndVersion(cfg, content); err != nil {
				handleError(err)
			}
		} else {
			err = format.Format(in, os.Stdout)
			if err != nil {
				handleError(fmt.Errorf("format failed: %w", err))
			}
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

	cfg := config.GetDefaultConfig(file)
	if backupDir != "" {
		cfg.BackupDir = backupDir
	}

	if verifyOnly {
		issues, err := editor.VerifyAndLint(cfg, rules.ActualFS{})
		if err != nil {
			handleError(err)
		}

		var issueCount int
		for _, s := range issues {
			if s.Level == rules.IssueLevelError || s.Level == rules.IssueLevelWarning || s.Level == rules.IssueLevelSecurity {
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
