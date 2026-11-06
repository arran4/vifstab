// Package cli binds generated commands to the verification and editing workflows.
package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/arran4/vifstab"
	"github.com/arran4/vifstab/cmd"
	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/diagram"
	"github.com/arran4/vifstab/internal/editor"
	"github.com/arran4/vifstab/internal/format"
	"github.com/arran4/vifstab/internal/outcome"
	"github.com/arran4/vifstab/internal/rules"
	"github.com/arran4/vifstab/internal/uefi"
	"github.com/cpuguy83/go-md2man/v2/md2man"
)

// Verify is a subcommand `vifstab verify` -- Verify fstab read-only with the caller's credentials.
//
// Flags:
//
//	file: --file (default: "/etc/fstab") File to verify.
//	seen: (flag presence) Explicitly supplied flags.
//	operands: ... No positional operands are accepted.
func Verify(file string, seen map[string]bool, operands ...string) error {
	if !seen["file"] {
		file = vifstab.DefaultFile
	}
	if err := noOperands("vifstab verify", operands); err != nil {
		return err
	}
	issues, err := editor.VerifyAndLint(config.Config{FilePath: file}, rules.ActualFS{})
	if err != nil {
		return applicationError(err)
	}
	return Report(os.Stdout, issues)
}

// Report prints verification findings and applies the shared blocking-severity policy.
func Report(w io.Writer, issues []rules.ValidationIssue) error {
	blocking := false
	for _, issue := range issues {
		blocking = blocking || rules.IsBlockingIssue(issue)
	}
	if blocking {
		if _, err := fmt.Fprintln(w, "Validation issues found:"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "Validation successful."); err != nil {
			return err
		}
		if len(issues) > 0 {
			if _, err := fmt.Fprintln(w, "Issues:"); err != nil {
				return err
			}
		}
	}
	for _, issue := range issues {
		if _, err := fmt.Fprintf(w, " - [%s] %s (Line: %s)\n", issue.Level, issue.Message, issue.Line); err != nil {
			return err
		}
	}
	if blocking {
		return &cmd.ErrExitCode{Code: 1}
	}
	return nil
}

// Edit is a subcommand `vifstab edit` -- Edit, validate and save fstab explicitly, using inherited credentials.
//
// Flags:
//
//	file: --file (default: "/etc/fstab") File to edit.
//	backupDir: --backup-dir (default: "") RCS backup directory; defaults to the caller's cache.
//	systemdReload: --systemd-reload (default: "warn") Post-save systemd handling: auto, warn or never.
//	seen: (flag presence) Explicitly supplied flags.
//	operands: ... No positional operands are accepted.
func Edit(file, backupDir, systemdReload string, seen map[string]bool, operands ...string) error {
	if !seen["file"] {
		file = vifstab.DefaultFile
	}
	if err := noOperands("vifstab edit", operands); err != nil {
		return err
	}
	mode := config.SystemdReloadAction(systemdReload)
	switch mode {
	case config.SystemdReloadAuto, config.SystemdReloadWarn, config.SystemdReloadNever:
	default:
		return fmt.Errorf("invalid --systemd-reload value %q; expected auto, warn or never", systemdReload)
	}
	cfg, err := writingConfig(file, backupDir)
	if err != nil {
		return err
	}
	cfg.SystemdReload = mode
	return applicationError(editor.EditAndLint(cfg, rules.ActualFS{}, editor.DefaultDependencies()))
}

func writingConfig(file, backupDir string) (config.Config, error) {
	cfg, err := config.GetDefaultConfig(file)
	if backupDir != "" {
		cfg.BackupDir = backupDir
		err = nil
	}
	return cfg, err
}

// Format is a subcommand `vifstab format` -- Print aligned fstab; writing requires --in-place.
//
// Flags:
//
//	file: --file (default: "/etc/fstab") Input file; an explicit path takes precedence over piped stdin. Use - for stdin.
//	inPlace: --in-place -i (default: false) Replace the selected file using the save and recovery workflow.
//	backupDir: --backup-dir (default: "") RCS directory for in-place formatting.
//	seen: (flag presence) Explicitly supplied flags.
//	operands: ... No positional operands are accepted.
func Format(file string, inPlace bool, backupDir string, seen map[string]bool, operands ...string) error {
	if !seen["file"] {
		file = vifstab.DefaultFile
	}
	if err := noOperands("vifstab format", operands); err != nil {
		return err
	}
	useStdin := file == "-"
	if !seen["file"] {
		var err error
		useStdin, err = stdinIsPiped(os.Stdin)
		if err != nil {
			return err
		}
	}
	if useStdin && inPlace {
		return fmt.Errorf("cannot use --in-place when reading from standard input")
	}
	var input io.Reader = os.Stdin
	if !useStdin {
		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", file, err)
		}
		defer func() { _ = f.Close() }()
		input = f
	}
	if !inPlace {
		return format.Format(input, os.Stdout)
	}
	var formatted bytes.Buffer
	if err := format.Format(input, &formatted); err != nil {
		return fmt.Errorf("format failed: %w", err)
	}
	cfg, err := writingConfig(file, backupDir)
	if err != nil {
		return err
	}
	return applicationError(editor.SaveWithoutValidation(cfg, formatted.Bytes(), editor.DefaultDependencies()))
}

func stdinIsPiped(input interface{ Stat() (os.FileInfo, error) }) (bool, error) {
	info, err := input.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to inspect standard input: %w", err)
	}
	return info.Mode()&os.ModeCharDevice == 0, nil
}

// Diagram is a subcommand `vifstab diagram` -- Print a partition diagram.
//
// Flags:
//
//	devices: @1 1...1 Device or disk image.
func Diagram(devices ...string) error { return diagram.Print(os.Stdout, devices[0]) }

// UEFI is a subcommand `vifstab uefi` -- Experimental live firmware correlation; unavailable observations remain unknown.
//
// Flags:
//
//	operands: @1 2...2 Device identity and mountpoint to correlate.
func UEFI(operands ...string) error {
	device, mountpoint := operands[0], operands[1]
	info, err := uefi.Analyze(rules.ActualFS{}, uefi.ActualEFIProvider{}, device, mountpoint)
	if err != nil {
		return err
	}
	fmt.Print(uefi.FormatReport(info, device))
	return nil
}

// Manpage is a subcommand `vifstab manpage` -- Generate the manual page explicitly.
//
// Flags:
//
//	output: --output (default: "vifstab.1") Destination file.
//	operands: ... No positional operands are accepted.
func Manpage(output string, operands ...string) error {
	if err := noOperands("vifstab manpage", operands); err != nil {
		return err
	}
	return os.WriteFile(output, md2man.Render(vifstab.ManpageMarkdown), 0644)
}

func applicationError(err error) error {
	var result *outcome.ApplicationOutcome
	if errors.As(err, &result) {
		return &cmd.ErrExitCode{Code: result.ExitCode, Err: result.Err}
	}
	return err
}

// noOperands validates action inputs before any I/O. Command routing remains
// entirely generated; gosubc's Go flag backend permits unbound operands.
func noOperands(path string, operands []string) error {
	if len(operands) == 0 {
		return nil
	}
	return &cmd.UsageError{CommandPath: path, Kind: cmd.UsageInvalidArguments, Message: fmt.Sprintf("unexpected arguments: %q", operands)}
}
