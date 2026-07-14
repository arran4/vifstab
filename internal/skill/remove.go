package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

func RemoveSkill(skillName string, agent string, scope string) error {
	if skillName == "" {
		return fmt.Errorf("skill name required for remove")
	}

	tr := NewTargetResolver(agent, scope)
	basePath, err := tr.ResolvePath()
	if err != nil {
		return err
	}

	destDir := filepath.Join(basePath, skillName)

	// Verify it's a skill directory by checking for metadata
	if _, err := os.Stat(filepath.Join(destDir, ".vifstab-skill.json")); os.IsNotExist(err) {
		return fmt.Errorf("skill '%s' not found or is not a valid installed skill", skillName)
	}

	err = os.RemoveAll(destDir)
	if err != nil {
		return fmt.Errorf("failed to remove skill directory: %w", err)
	}

	fmt.Printf("Successfully removed skill '%s'\n", skillName)
	return nil
}
