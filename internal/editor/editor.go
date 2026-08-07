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
)

func EditAndLint(cfg config.Config, api rules.API) error {
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

	editor := os.Getenv("SUDO_EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}

	var editorParts []string
	if editor != "" {
		var err error
		editorParts, err = SplitEditorCommand(editor)
		if err != nil {
			return fmt.Errorf("failed to parse editor command: %w", err)
		}
	}

	if len(editorParts) == 0 {
		editorParts = []string{"vi"}
	}

	for _, part := range editorParts {
		if part == "--" {
			return fmt.Errorf("editor arguments may not contain \"--\"")
		}
	}

	for {
		// 2. Open Editor
		cmdArgs := append(editorParts[1:], "--", tempFile.Name())
		cmd := exec.Command(editorParts[0], cmdArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
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

		if len(allIssues) == 0 {
			break // Valid
		}

		// 4. Invalid, Prompt user
		fmt.Println("Validation issues found:")
		for _, s := range allIssues {
			fmt.Printf(" - [%s] %s (Line: %s)\n", s.Level, s.Message, s.Line)
		}

		fmt.Print("What now? (e)dit again, (c)ontinue and save anyway, (q)uit without saving: ")
		scanner := bufio.NewScanner(os.Stdin)
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

	if err := SaveAndVersion(cfg, content); err != nil {
		return err
	}

	fmt.Println("Saved successfully.")
	return nil
}

// SaveAndVersion saves the given content to the target file and versions it using RCS.
func SaveAndVersion(cfg config.Config, content []byte) error {
	// First update RCS
	rcsCfg := rcs.Config{Dir: cfg.BackupDir}
	if err := rcs.Update(cfg.FilePath, content, rcsCfg); err != nil {
		return fmt.Errorf("rcs update failed: %w", err)
	}

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
	if err := tmpDest.Close(); err != nil {
		return fmt.Errorf("failed to close temporary destination file: %w", err)
	}

	if err := os.Rename(tmpDest.Name(), cfg.FilePath); err != nil {
		return fmt.Errorf("failed to replace destination file: %w", err)
	}

	return nil
}
