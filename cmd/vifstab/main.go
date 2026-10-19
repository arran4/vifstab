package main

import (
	"context"
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
	"github.com/arran4/vifstab/internal/txtar"
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

	if len(os.Args) > 1 && os.Args[1] == "capture" {
		err := runCapture(os.Args[2:])
		if err != nil {
			handleError(err)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "replay" {
		err := runReplay(os.Args[2:])
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
	var txtarFile string
	flag.StringVar(&txtarFile, "txtar", "", "Run validation against a captured TXTAR snapshot instead of live system")
	flag.StringVar(&systemdReload, "systemd-reload", "warn", "Post-save systemd handling for /etc/fstab (auto|warn|never)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	if txtarFile != "" && !verifyOnly {
		handleError(fmt.Errorf("--txtar requires --verify; use 'vifstab replay' for offline validation"))
	}

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
		if txtarFile != "" {
			err := runVerifyTxtar(txtarFile)
			if err != nil {
				handleError(err)
			}
			return
		}
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

func runCapture(args []string) error {
	captureCmd := flag.NewFlagSet("capture", flag.ExitOnError)
	fileFlag := captureCmd.String("file", "/etc/fstab", "File to capture")
	unredactedFlag := captureCmd.Bool("unredacted", false, "Do not redact sensitive values")
	captureCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: vifstab capture [options] [<output.txtar>]\n")
		captureCmd.PrintDefaults()
	}
	_ = captureCmd.Parse(args)

	if captureCmd.NArg() > 1 {
		captureCmd.Usage()
		return fmt.Errorf("unexpected arguments: %v", captureCmd.Args()[1:])
	}

	outData, err := txtar.CaptureSystem(context.Background(), rules.ActualFS{}, *fileFlag, *unredactedFlag)
	if err != nil {
		return fmt.Errorf("capture failed: %w", err)
	}

	if captureCmd.NArg() == 1 {
		if err := os.WriteFile(captureCmd.Arg(0), outData, 0600); err != nil {
			return fmt.Errorf("failed to write capture: %w", err)
		}
	} else {
		os.Stdout.Write(outData)
	}
	return nil
}

func runReplay(args []string) error {
	replayCmd := flag.NewFlagSet("replay", flag.ExitOnError)
	replayCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: vifstab replay <snapshot.txtar>\n")
		replayCmd.PrintDefaults()
	}
	_ = replayCmd.Parse(args)

	if replayCmd.NArg() != 1 {
		replayCmd.Usage()
		return fmt.Errorf("expected 1 argument, got %d", replayCmd.NArg())
	}

	return runVerifyTxtar(replayCmd.Arg(0))
}

func runVerifyTxtar(txtarPath string) error {
	data, err := os.ReadFile(txtarPath)
	if err != nil {
		return fmt.Errorf("failed to read txtar file: %w", err)
	}

	snapshot, err := txtar.ParseTXTAR(data)
	if err != nil {
		return fmt.Errorf("failed to parse txtar file: %w", err)
	}

	api, err := txtar.ReplaySnapshot(snapshot)
	if err != nil {
		return fmt.Errorf("failed to setup replay environment: %w", err)
	}

	tempFstab, err := os.CreateTemp("", "fstab-replay-*")
	if err != nil {
		return err
	}
	defer os.Remove(tempFstab.Name())
	tempFstab.Write(snapshot.Fstab)
	tempFstab.Close()

	cfg, err := config.GetDefaultConfig(tempFstab.Name())
	if err != nil {
		return err
	}

	issues, err := editor.VerifyAndLint(cfg, api)
	if err != nil {
		return err
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
		return nil
	}

	fmt.Println("Validation issues found:")
	for _, s := range issues {
		fmt.Printf(" - [%s] %s (Line: %s)\n", s.Level, s.Message, s.Line)
	}

	return &outcome.ApplicationOutcome{
		ExitCode: 1,
		Err:      fmt.Errorf("validation failed with %d blocking issues", issueCount),
	}
}
