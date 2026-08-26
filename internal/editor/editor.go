package editor

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/rcs"
	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin" // Register rules
	"github.com/arran4/vifstab/internal/rules/core"
	"github.com/arran4/vifstab/internal/systemd"
)

// FS provides filesystem operations to allow mocking
type FS interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Rename(oldpath, newpath string) error
	Stat(name string) (os.FileInfo, error)
}

type osFS struct{}

func (osFS) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (osFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (osFS) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (osFS) Rename(oldpath, newpath string) error         { return os.Rename(oldpath, newpath) }
func (osFS) Stat(name string) (os.FileInfo, error)        { return os.Stat(name) }

var DefaultFS FS = osFS{}

// Dependencies contains the side-effecting services used by the edit workflow.
// Supplying it explicitly keeps tests on the same production path as the CLI.
type Dependencies struct {
	FS      FS
	Systemd systemd.Manager
	Editor  func(path string) error
	Input   io.Reader
}

func DefaultDependencies() Dependencies {
	return Dependencies{FS: DefaultFS, Systemd: systemd.NewManager(nil)}
}

func (d Dependencies) withDefaults() Dependencies {
	if d.FS == nil {
		d.FS = DefaultFS
	}
	if d.Systemd == nil {
		d.Systemd = systemd.NewManager(nil)
	}
	if d.Input == nil {
		d.Input = os.Stdin
	}
	return d
}

func EditAndLint(cfg config.Config, api rules.API, deps Dependencies) error {
	deps = deps.withDefaults()

	// 1. Copy target file to temp
	tempFile, err := os.CreateTemp("", "vifstab-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()
	defer func() { _ = tempFile.Close() }()

	src, err := os.Open(cfg.FilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to open source file: %w", err)
		}
	} else {
		_, err = io.Copy(tempFile, src)
		_ = src.Close()
		if err != nil {
			return fmt.Errorf("failed to copy to temp file: %w", err)
		}
	}
	_ = tempFile.Close()

	if deps.Editor == nil {
		editorParts, err := editorCommandFromEnvironment()
		if err != nil {
			return err
		}
		deps.Editor = func(path string) error {
			cmdArgs := append(editorParts[1:], "--", path)
			cmd := exec.Command(editorParts[0], cmdArgs...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
	}

	blockingValidationPassed := false
	for {
		// 2. Open Editor
		if err := deps.Editor(tempFile.Name()); err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}

		// 3. Read and Lint
		f, err := os.Open(tempFile.Name())
		if err != nil {
			return fmt.Errorf("failed to read edited file: %w", err)
		}

		entries, err := core.ParseFstab(f)
		_ = f.Close()
		if err != nil {
			return fmt.Errorf("failed to parse: %w", err)
		}

		// Run rules
		allIssues, err := lintEntries(entries, api)
		if err != nil {
			return err
		}

		var issueCount int
		for _, s := range allIssues {
			if rules.IsBlockingIssue(s) {
				issueCount++
			}
		}

		if issueCount == 0 {
			blockingValidationPassed = true
			if len(allIssues) > 0 {
				fmt.Println("Validation successful, but issues were noted:")
				for _, s := range allIssues {
					fmt.Printf(" - [%s] %s (Line: %s)\n", s.Level, s.Message, s.Line)
				}
			}
			break // Valid
		}

		// 4. Invalid, Prompt user
		fmt.Println("Validation issues found:")
		for _, s := range allIssues {
			fmt.Printf(" - [%s] %s (Line: %s)\n", s.Level, s.Message, s.Line)
		}

		fmt.Print("What now? (e)dit again, (c)ontinue and save anyway, (q)uit without saving: ")
		scanner := bufio.NewScanner(deps.Input)
		if !scanner.Scan() {
			return fmt.Errorf("failed to read user input: %w", scanner.Err())
		}
		choice := scanner.Text()

		if choice == "c" || choice == "C" {
			break
		} else if choice == "q" || choice == "Q" {
			fmt.Println("Aborted.")
			return nil
		}
		// otherwise loop back to edit
	}

	// 5. Save and Version
	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to read temp file for saving: %w", err)
	}

	return commitEdit(cfg, content, blockingValidationPassed, deps)
}

func editorCommandFromEnvironment() ([]string, error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}

	var editorParts []string
	if editor != "" {
		var err error
		editorParts, err = SplitEditorCommand(editor)
		if err != nil {
			return nil, fmt.Errorf("failed to parse editor command: %w", err)
		}
	}

	if len(editorParts) == 0 {
		editorParts = []string{"vi"}
	}

	for _, part := range editorParts {
		if part == "--" {
			return nil, fmt.Errorf("editor arguments may not contain \"--\"")
		}
	}

	return editorParts, nil
}

// SaveAndVersion saves the given content to the target file and versions it using RCS.
// It deliberately has no post-save systemd side effects.
func SaveAndVersion(cfg config.Config, content []byte, fsLayer FS) error {
	if fsLayer == nil {
		fsLayer = DefaultFS
	}
	// First update RCS
	rcsCfg := rcs.Config{Dir: cfg.BackupDir}
	if err := rcs.Update(cfg.FilePath, content, rcsCfg, fsLayer); err != nil {
		return fmt.Errorf("rcs update failed: %w", err)
	}

	var targetMode *os.FileMode
	if info, err := fsLayer.Stat(cfg.FilePath); err == nil {
		m := info.Mode().Perm()
		targetMode = &m
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat destination file: %w", err)
	}

	if _, isOSFS := fsLayer.(osFS); isOSFS {
		// Then copy to dest atomically
		dir := filepath.Dir(cfg.FilePath)
		tmpDest, err := os.CreateTemp(dir, "vifstab-tmp-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary destination file: %w", err)
		}
		defer func() { _ = os.Remove(tmpDest.Name()) }()

		if _, err := tmpDest.Write(content); err != nil {
			_ = tmpDest.Close()
			return fmt.Errorf("failed to write to temporary destination file: %w", err)
		}
		if targetMode != nil {
			if err := tmpDest.Chmod(*targetMode); err != nil {
				_ = tmpDest.Close()
				return fmt.Errorf("failed to chmod temporary destination file: %w", err)
			}
		}
		if err := tmpDest.Close(); err != nil {
			return fmt.Errorf("failed to close temporary destination file: %w", err)
		}

		if err := fsLayer.Rename(tmpDest.Name(), cfg.FilePath); err != nil {
			return fmt.Errorf("failed to replace destination file: %w", err)
		}
	} else {
		// Write the destination directly
		perm := os.FileMode(0600)
		if targetMode != nil {
			perm = *targetMode
		}
		if err := fsLayer.WriteFile(cfg.FilePath, content, perm); err != nil {
			return fmt.Errorf("failed to replace destination file: %w", err)
		}
	}

	return nil
}

// commitEdit commits an edited fstab before considering any systemd action.
// A user may save after blocking validation fails, but that save is never
// eligible for automatic reload handling.
func commitEdit(cfg config.Config, content []byte, blockingValidationPassed bool, deps Dependencies) error {
	deps = deps.withDefaults()
	if err := SaveAndVersion(cfg, content, deps.FS); err != nil {
		return err
	}
	fmt.Println("Saved successfully.")
	if !blockingValidationPassed {
		return nil
	}
	return HandleSystemdReload(cfg, deps.Systemd)
}

// HandleSystemdReload performs the explicitly configured post-save action.
func HandleSystemdReload(cfg config.Config, sm systemd.Manager) error {
	if cfg.FilePath != "/etc/fstab" || !sm.IsActive() {
		return nil
	}

	switch cfg.SystemdReload {
	case config.SystemdReloadAuto:
		if err := sm.Reload(); err != nil {
			return fmt.Errorf("fstab saved, but systemd daemon-reload failed: %w", err)
		}
		fmt.Println("note: systemd daemon-reload completed.")
	case config.SystemdReloadWarn, "":
		fmt.Println("warning: systemd mount-unit state predates this fstab change.")
		fmt.Println("run: systemctl daemon-reload")
	case config.SystemdReloadNever:
		// Do nothing.
	}
	return nil
}
