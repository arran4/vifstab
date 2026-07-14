package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

func InspectSkill(skillName string, agent string, scope string) error {
	if skillName == "" {
		return fmt.Errorf("skill name required for inspect")
	}

	tr := NewTargetResolver(agent, scope)
	basePath, err := tr.ResolvePath()
	if err != nil {
		return err
	}

	destDir := filepath.Join(basePath, skillName)

	meta, err := readMetadata(destDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("skill '%s' not found or is not a valid installed skill", skillName)
		}
		return fmt.Errorf("failed to read skill metadata: %w", err)
	}

	fmt.Printf("Name:        %s\n", meta.Name)
	fmt.Printf("Source Type: %s\n", meta.SourceType)
	fmt.Printf("Source URL:  %s\n", meta.SourceURL)
	if meta.SourcePath != "" {
		fmt.Printf("Source Path: %s\n", meta.SourcePath)
	}
	fmt.Printf("Revision:    %s\n", meta.Revision)
	fmt.Printf("Install Dir: %s\n", destDir)
	fmt.Printf("Target Agent:%s\n", meta.TargetAgent)
	fmt.Printf("Scope:       %s\n", meta.Scope)
	fmt.Printf("Install Time:%s\n", meta.InstallTime.Format("2006-01-02 15:04:05"))

	// Check local modifications
	currentHash, err := hashDir(destDir)
	if err == nil {
		if currentHash != meta.LocalModHash {
			fmt.Printf("Status:      MODIFIED LOCALLY\n")
		} else {
			fmt.Printf("Status:      Clean\n")
		}
	} else {
		fmt.Printf("Status:      Error checking hash (%v)\n", err)
	}

	// Print files
	fmt.Printf("\nFiles:\n")
	filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(destDir, path)
			fmt.Printf("  - %s\n", rel)
		}
		return nil
	})

	return nil
}
