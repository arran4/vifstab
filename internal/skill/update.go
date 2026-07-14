package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

func UpdateSkill(skillName string, agent string, scope string, force bool) error {
	if skillName == "" {
		return fmt.Errorf("skill name required for update")
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

	if meta.SourceType == "local" || meta.SourceType == "embedded" {
		fmt.Printf("Skill '%s' is from a local or embedded source. Re-installing...\n", skillName)
		return InstallSkill(meta.SourceURL, skillName, agent, scope)
	}

	source, _, _ := parseSource(meta.SourceURL)
	if source == nil {
		return fmt.Errorf("could not parse original source URL: %s", meta.SourceURL)
	}

	gh, ok := source.(*GitHubSource)
	if !ok {
		return fmt.Errorf("unsupported source type for update: %s", meta.SourceType)
	}
	gh.Path = meta.SourcePath

	// Check local modifications
	currentHash, err := hashDir(destDir)
	if err == nil {
		if currentHash != meta.LocalModHash && !force {
			return fmt.Errorf("installed skill has local modifications; rerun with --force to replace")
		}
	}

	// Check for updates
	fmt.Printf("Checking for updates for '%s'...\n", skillName)
	latestSha, err := gh.getLatestCommitSHA()
	if err != nil {
		return fmt.Errorf("failed to check upstream revision: %w", err)
	}

	if latestSha == meta.Revision && !force {
		fmt.Printf("Skill '%s' is already up-to-date (revision: %s).\n", skillName, latestSha)
		return nil
	}

	fmt.Printf("Updating skill '%s' to revision %s...\n", skillName, latestSha)

	// Create a temp directory for the new skill to ensure atomicity
	tmpDir := destDir + ".update-tmp"
	os.RemoveAll(tmpDir) // clean if exists
	err = os.MkdirAll(tmpDir, 0755)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	newRev, err := source.FetchTo(tmpDir)
	if err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "SKILL.md")); os.IsNotExist(err) {
		return fmt.Errorf("update is invalid: SKILL.md not found in upstream")
	}

	// Calculate new hash
	newHash, err := hashDir(tmpDir)
	if err != nil {
		return err
	}

	meta.Revision = newRev
	meta.LocalModHash = newHash
	err = writeMetadata(tmpDir, meta)
	if err != nil {
		return err
	}

	// Swap directories atomically (or as close as we can get)
	err = os.RemoveAll(destDir)
	if err != nil {
		return fmt.Errorf("failed to remove old skill directory: %w", err)
	}

	err = os.Rename(tmpDir, destDir)
	if err != nil {
		return fmt.Errorf("failed to replace skill directory: %w", err)
	}

	fmt.Printf("Successfully updated skill '%s'\n", skillName)
	return nil
}
