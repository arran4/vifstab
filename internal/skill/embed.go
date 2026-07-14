package skill

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed embedded_skills/vifstab
var embeddedSkillsFS embed.FS

type EmbeddedSource struct{}

func (s *EmbeddedSource) FetchTo(destDir string) (string, error) {
	err := fs.WalkDir(embeddedSkillsFS, "embedded_skills/vifstab", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel("embedded_skills/vifstab", path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := fs.ReadFile(embeddedSkillsFS, path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0644)
	})
	if err != nil {
		return "", fmt.Errorf("failed to extract embedded skill: %w", err)
	}

	return "embedded-v1", nil
}
