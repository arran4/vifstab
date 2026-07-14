package skill

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SkillSource interface {
	FetchTo(destDir string) (string, error) // Returns revision, error
}

func parseSource(sourceArg string) (SkillSource, string, string) {
	// Simple heuristic
	if strings.HasPrefix(sourceArg, "./") || strings.HasPrefix(sourceArg, "/") || strings.HasPrefix(sourceArg, "../") {
		// Local source
		return &LocalSource{Path: sourceArg}, "local", sourceArg
	} else if sourceArg == "vifstab" {
		// Embedded
		return &EmbeddedSource{}, "embedded", "vifstab"
	}

	// Assume GitHub owner/repo[/path]
	parts := strings.Split(sourceArg, "/")
	if len(parts) >= 2 {
		gh := &GitHubSource{
			Owner: parts[0],
			Repo:  parts[1],
		}
		if len(parts) > 2 {
			gh.Path = strings.Join(parts[2:], "/")
		}
		return gh, "github", sourceArg
	}

	return nil, "", ""
}

func InstallSkill(sourceArg string, skillName string, agent string, scope string) error {
	source, srcType, srcURL := parseSource(sourceArg)
	if source == nil {
		return fmt.Errorf("invalid source format: %s", sourceArg)
	}

	// Use repository name if skillName not provided for GitHub
	if skillName == "" {
		if srcType == "github" {
			gh := source.(*GitHubSource)
			skillName = gh.Repo
		} else if srcType == "local" {
			skillName = filepath.Base(sourceArg)
		} else if srcType == "embedded" {
			skillName = "vifstab"
		}
	}

	tr := NewTargetResolver(agent, scope)
	basePath, err := tr.ResolvePath()
	if err != nil {
		return err
	}

	destDir := filepath.Join(basePath, skillName)

	// Clean up partial installs if we fail
	defer func() {
		if err != nil {
			// but don't delete if we are updating, handled separately.
			// For fresh install, delete:
			if _, e := os.Stat(filepath.Join(destDir, ".vifstab-skill.json")); os.IsNotExist(e) {
				os.RemoveAll(destDir)
			}
		}
	}()

	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	revision, err := source.FetchTo(destDir)
	if err != nil {
		return err
	}

	// Validate SKILL.md exists
	if _, err := os.Stat(filepath.Join(destDir, "SKILL.md")); os.IsNotExist(err) {
		return fmt.Errorf("skill is invalid: SKILL.md not found in source")
	}

	hash, err := hashDir(destDir)
	if err != nil {
		return err
	}

	meta := SkillMetadata{
		Name:         skillName,
		SourceType:   srcType,
		SourceURL:    srcURL,
		SourcePath:   "",
		Revision:     revision,
		InstallTime:  time.Now(),
		TargetAgent:  agent,
		Scope:        scope,
		LocalModHash: hash,
	}

	if srcType == "github" {
		gh := source.(*GitHubSource)
		meta.SourcePath = gh.Path
	}

	err = writeMetadata(destDir, meta)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully installed skill '%s' to %s\n", skillName, destDir)
	return nil
}

func writeMetadata(dir string, meta SkillMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ".vifstab-skill.json"), data, 0644)
}

func readMetadata(dir string) (SkillMetadata, error) {
	var meta SkillMetadata
	data, err := os.ReadFile(filepath.Join(dir, ".vifstab-skill.json"))
	if err != nil {
		return meta, err
	}
	err = json.Unmarshal(data, &meta)
	return meta, err
}

func hashDir(dir string) (string, error) {
	h := sha256.New()
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path) == ".vifstab-skill.json" {
			return nil // ignore metadata file itself
		}

		rel, _ := filepath.Rel(dir, path)
		h.Write([]byte(rel))

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		io.Copy(h, f)
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
