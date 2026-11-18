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
	"github.com/arran4/vifstab/internal/textinput"
	"github.com/arran4/vifstab/internal/uefi"
	"github.com/cpuguy83/go-md2man/v2/md2man"
)

//go:embed manpage.md
var manPageMarkdown []byte

func main() {
	if len(os.Args) > 1 && os.Args[1] == "uefi" {
		uefiCmd := flag.NewFlagSet("uefi", flag.ExitOnError)
		uefiCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: %s uefi <device> <mountpoint>\n", os.Args[0])
			uefiCmd.PrintDefaults()
		}
		_ = uefiCmd.Parse(os.Args[2:])

		if uefiCmd.NArg() != 2 {
			uefiCmd.Usage()
			os.Exit(1)
		}

		device := uefiCmd.Arg(0)
		mountPoint := uefiCmd.Arg(1)

		info, err := uefi.Analyze(rules.ActualFS{}, uefi.ActualEFIProvider{}, device, mountPoint)
		if err != nil {
			handleError(err)
		}

		fmt.Print(uefi.FormatReport(info, device))
		return
	}

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
		backupDirFlag := formatCmd.String("backup-dir", "", "Directory to store RCS backups")
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

		var in io.ReadCloser
		var err error

		// If formatting standard input
		piped, err := stdinIsPiped(os.Stdin)
		if err != nil {
			handleError(err)
		}
		if piped {
			in = os.Stdin
			if err := textinput.Check(os.Stdin); err != nil {
				handleError(err)
			}
			if *inPlaceFlag {
				handleError(fmt.Errorf("cannot use -i when reading from standard input"))
			}
		} else {
			in, err = textinput.Open(*fileFlag)
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

			cfg, err := config.GetDefaultConfig(*fileFlag)
			if err != nil && *backupDirFlag == "" {
				handleError(err)
			}
			if *backupDirFlag != "" {
				cfg.BackupDir = *backupDirFlag
			}
			if err := editor.SaveWithoutValidation(cfg, content, editor.DefaultDependencies()); err != nil {
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
		systemdReload   string
	)

	flag.StringVar(&backupDir, "backup-dir", "", "Directory to store RCS backups (defaults to a sensible location based on the file being edited and the user running the command)")
	flag.StringVar(&file, "file", "/etc/fstab", "File to edit")
	flag.BoolVar(&generateManpage, "generate-manpage", false, "Generate man page for vifstab to vifstab.1")
	flag.BoolVar(&verifyOnly, "verify", false, "Verify the file without opening an editor")
	flag.StringVar(&systemdReload, "systemd-reload", "warn", "Post-save systemd handling for /etc/fstab (auto|warn|never)")

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

	cfg, err := config.GetDefaultConfig(file)

	if backupDir != "" {
		cfg.BackupDir = backupDir
	}

	switch config.SystemdReloadAction(systemdReload) {
	case config.SystemdReloadAuto, config.SystemdReloadWarn, config.SystemdReloadNever:
		cfg.SystemdReload = config.SystemdReloadAction(systemdReload)
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid --systemd-reload value '%s'. Allowed values are: auto, warn, never\n", systemdReload)
		os.Exit(1)
	}

	if verifyOnly {
		issues, err := editor.VerifyAndLint(cfg, rules.ActualFS{})
		if err != nil {
			handleError(err)
		}

		var issueCount int
		for _, s := range issues {
			if rules.IsBlockingIssue(s) {
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

	if err != nil && backupDir == "" {
		handleError(err)
	}

	err = editor.EditAndLint(cfg, rules.ActualFS{}, editor.DefaultDependencies())
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

// stdinIsPiped distinguishes terminal input from a pipe or regular file.
// A failed probe is an operational error, never evidence that stdin is a pipe.
func stdinIsPiped(input interface{ Stat() (os.FileInfo, error) }) (bool, error) {
	info, err := input.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to inspect standard input: %w", err)
	}
	return info.Mode()&os.ModeCharDevice == 0, nil
}
