package editor

import (
	"bufio"
	"context"
	"errors"
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
	"github.com/arran4/vifstab/internal/textinput"
)

type TempFile interface {
	io.WriteCloser
	Name() string
	Chmod(mode os.FileMode) error
	Sync() error
}

// FS provides filesystem operations to allow mocking
type FS interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Rename(oldpath, newpath string) error
	Stat(name string) (os.FileInfo, error)
	Lstat(name string) (os.FileInfo, error)
	EvalSymlinks(path string) (string, error)
	CreateTemp(dir, pattern string) (TempFile, error)
	Remove(name string) error
	SyncDir(name string) error
}

type osFS struct{}

func (osFS) ReadFile(name string) ([]byte, error) { return textinput.ReadFile(name) }
func (osFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (osFS) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (osFS) Rename(oldpath, newpath string) error         { return os.Rename(oldpath, newpath) }
func (osFS) Stat(name string) (os.FileInfo, error)        { return os.Stat(name) }
func (osFS) Lstat(name string) (os.FileInfo, error)       { return os.Lstat(name) }
func (osFS) EvalSymlinks(path string) (string, error)     { return filepath.EvalSymlinks(path) }
func (osFS) CreateTemp(dir, pattern string) (TempFile, error) {
	return os.CreateTemp(dir, pattern)
}
func (osFS) Remove(name string) error { return os.Remove(name) }
func (osFS) SyncDir(name string) error {
	d, err := os.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}

var DefaultFS FS = osFS{}

type rcsFSAdapter struct {
	fs FS
}

func (a rcsFSAdapter) MkdirAll(path string, perm os.FileMode) error {
	return a.fs.MkdirAll(path, perm)
}

func (a rcsFSAdapter) Stat(name string) (os.FileInfo, error) {
	return a.fs.Stat(name)
}

func (a rcsFSAdapter) ReadFile(name string) ([]byte, error) {
	return a.fs.ReadFile(name)
}

func (a rcsFSAdapter) Rename(oldpath, newpath string) error {
	return a.fs.Rename(oldpath, newpath)
}

func (a rcsFSAdapter) Remove(name string) error {
	return a.fs.Remove(name)
}

func (a rcsFSAdapter) WriteFile(name string, data []byte, perm os.FileMode) error {
	return a.fs.WriteFile(name, data, perm)
}

func (a rcsFSAdapter) CreateTemp(dir, pattern string) (rcs.TempFile, error) {
	return a.fs.CreateTemp(dir, pattern)
}

func (a rcsFSAdapter) SyncDir(name string) error {
	return a.fs.SyncDir(name)
}

var _ rcs.FS = rcsFSAdapter{}

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

	source, err := deps.FS.ReadFile(cfg.FilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to open source file: %w", err)
		}
	} else {
		_, err = tempFile.Write(source)
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
		f, err := textinput.Open(tempFile.Name())
		if err != nil {
			return fmt.Errorf("failed to read edited file: %w", err)
		}

		entries, err := core.ParseFstab(f)
		_ = f.Close()
		if err != nil {
			return fmt.Errorf("failed to parse: %w", err)
		}

		// Run rules
		ctx := rules.WithTargetFstab(context.Background(), cfg.FilePath)
		allIssues, err := lintEntriesWithContext(ctx, entries, api)
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
	content, err := textinput.ReadFile(tempFile.Name())
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

// SaveResult captures the success or failure of distinct stages in the save pipeline.
type SaveResult struct {
	Saved             bool
	VersioningEnabled bool
	Versioned         bool
	Err               error
}

// SaveAndVersion saves the given content to the target file and versions it using RCS.
// It deliberately has no post-save systemd side effects.
//
// Durability guarantee:
// On supported systems, this function provides crash durability against sudden power loss.
// It writes to a temporary file, applies metadata, syncs the file data to disk, closes it,
// atomically renames it over the target, and finally syncs the containing directory to
// ensure the directory entry metadata is persisted.
func SaveAndVersion(cfg config.Config, content []byte, fsLayer FS) SaveResult {
	if fsLayer == nil {
		fsLayer = DefaultFS
	}

	var res SaveResult

	target, err := resolveTarget(cfg.FilePath, fsLayer)
	if err != nil {
		res.Err = err
		return res
	}

	dir := filepath.Dir(target.path)
	tmpDest, err := fsLayer.CreateTemp(dir, "vifstab-tmp-*")
	if err != nil {
		res.Err = fmt.Errorf("failed to create temporary destination file: %w", err)
		return res
	}
	defer func() { _ = fsLayer.Remove(tmpDest.Name()) }()

	if _, err := tmpDest.Write(content); err != nil {
		_ = tmpDest.Close()
		res.Err = fmt.Errorf("failed to write to temporary destination file: %w", err)
		return res
	}
	if target.info != nil {
		if err := tmpDest.Chmod(target.info.Mode().Perm()); err != nil {
			_ = tmpDest.Close()
			res.Err = fmt.Errorf("failed to chmod temporary destination file: %w", err)
			return res
		}
	}

	if err := tmpDest.Sync(); err != nil {
		_ = tmpDest.Close()
		res.Err = fmt.Errorf("failed to sync temporary destination file: %w", err)
		return res
	}

	if err := tmpDest.Close(); err != nil {
		res.Err = fmt.Errorf("failed to close temporary destination file: %w", err)
		return res
	}

	// Refuse a detected link retargeting while preparing the replacement. This
	// is an optimistic check, not protection from hostile directory races.
	current, err := resolveTarget(cfg.FilePath, fsLayer)
	if err != nil {
		res.Err = fmt.Errorf("failed to recheck destination before replacement: %w", err)
		return res
	}
	if current.path != target.path {
		res.Err = fmt.Errorf("destination symlink target changed; reconcile and retry")
		return res
	}
	if err := fsLayer.Rename(tmpDest.Name(), target.path); err != nil {
		res.Err = fmt.Errorf("failed to replace destination file: %w", err)
		return res
	}

	res.Saved = true

	if err := fsLayer.SyncDir(dir); err != nil {
		res.Err = errors.Join(res.Err, fmt.Errorf("failed to sync directory after rename: %w", err))
	}

	if cfg.BackupDir != "" {
		res.VersioningEnabled = true
		rcsCfg := rcs.Config{Dir: cfg.BackupDir}
		if err := rcs.Update(cfg.FilePath, content, rcsCfg, rcsFSAdapter{fs: fsLayer}); err != nil {
			res.Err = errors.Join(res.Err, fmt.Errorf("rcs update failed: %w", err))
			return res
		}
		res.Versioned = true
	}

	return res
}

// commitEdit commits an edited fstab before considering any systemd action.
// A user may save after blocking validation fails, but that save is never
// eligible for an automatic reload and receives the conservative warning.
func commitEdit(cfg config.Config, content []byte, blockingValidationPassed bool, deps Dependencies) error {
	deps = deps.withDefaults()
	res := SaveAndVersion(cfg, content, deps.FS)

	var sysErr error
	if res.Saved {
		fmt.Println("Saved successfully.")
		sysErr = handlePostSaveSystemd(cfg, deps.Systemd, blockingValidationPassed)
	}

	return errors.Join(res.Err, sysErr)
}

// SaveWithoutValidation saves content produced by a workflow that has not
// passed blocking editor validation, then conservatively reports stale systemd
// state. It never performs an automatic reload.
func SaveWithoutValidation(cfg config.Config, content []byte, deps Dependencies) error {
	deps = deps.withDefaults()
	res := SaveAndVersion(cfg, content, deps.FS)

	var sysErr error
	if res.Saved {
		sysErr = handlePostSaveSystemd(cfg, deps.Systemd, false)
	}

	return errors.Join(res.Err, sysErr)
}

// HandleSystemdReload performs the configured post-save action after
// successful blocking validation.
func HandleSystemdReload(cfg config.Config, sm systemd.Manager) error {
	return handlePostSaveSystemd(cfg, sm, true)
}

func handlePostSaveSystemd(cfg config.Config, sm systemd.Manager, autoEligible bool) error {
	if cfg.FilePath != "/etc/fstab" || !sm.IsActive() {
		return nil
	}

	switch cfg.SystemdReload {
	case config.SystemdReloadAuto:
		if !autoEligible {
			printStaleSystemdWarning()
			return nil
		}
		if err := sm.Reload(); err != nil {
			return fmt.Errorf("fstab saved, but systemd daemon-reload failed: %w", err)
		}
		fmt.Println("note: systemd daemon-reload completed.")
	case config.SystemdReloadWarn, "":
		printStaleSystemdWarning()
	case config.SystemdReloadNever:
		// Do nothing.
	}
	return nil
}

func printStaleSystemdWarning() {
	fmt.Println("warning: systemd mount-unit state predates this fstab change.")
	fmt.Println("run: systemctl daemon-reload")
	fmt.Println("note: currently mounted filesystems may also require a remount or reboot.")
}
