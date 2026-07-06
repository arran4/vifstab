package editor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/arran4/vifstab/internal/config"
	"github.com/arran4/vifstab/internal/rcs"
	"github.com/arran4/vifstab/internal/rules"
	_ "github.com/arran4/vifstab/internal/rules/builtin" // Register rules
	"github.com/arran4/vifstab/internal/rules/core"
)

func EditAndLint(cfg config.Config) error {
	// 1. Copy target file to temp
	tempFile, err := os.CreateTemp("", "vifstab-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	src, err := os.Open(cfg.FilePath)
	if err == nil {
		_, err = io.Copy(tempFile, src)
		src.Close()
		if err != nil {
			return fmt.Errorf("failed to copy to temp file: %w", err)
		}
	}
	tempFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	for {
		// 2. Open Editor
		cmd := exec.Command(editor, tempFile.Name())
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
		f.Close()
		if err != nil {
			return fmt.Errorf("failed to parse: %w", err)
		}

		// Run rules
		allRules := rules.GetAllRules()
		ctx := rules.WithCache(context.Background(), rules.NewCache())

		var allSuggestions []rules.Suggestion
		for _, entry := range entries {
			for _, r := range allRules {
				suggestions := r.Validate(ctx, entry)
				allSuggestions = append(allSuggestions, suggestions...)
			}
		}

		if len(allSuggestions) == 0 {
			break // Valid
		}

		// 4. Invalid, Prompt user
		fmt.Println("Validation errors found:")
		for _, s := range allSuggestions {
			fmt.Printf(" - %s (Line: %s)\n", s.Message, s.Line)
		}

		fmt.Print("What now? (e)dit again, (c)ontinue and save anyway, (q)uit without saving: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
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

	// First update RCS
	rcsCfg := rcs.Config{Dir: cfg.BackupDir}
	if err := rcs.Update(cfg.FilePath, content, rcsCfg); err != nil {
		return fmt.Errorf("rcs update failed: %w", err)
	}

	// Then copy to dest
	if err := os.WriteFile(cfg.FilePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	fmt.Println("Saved successfully.")
	return nil
}
