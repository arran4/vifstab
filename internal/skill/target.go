package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

// TargetResolver handles the resolution of skill installation directories.
type TargetResolver struct {
	Agent string
	Scope string
}

func NewTargetResolver(agent, scope string) *TargetResolver {
	if agent == "" {
		agent = "common"
	}
	if scope == "" {
		scope = "user"
	}
	return &TargetResolver{
		Agent: agent,
		Scope: scope,
	}
}

// ResolvePath returns the absolute directory path where skills should be stored.
func (tr *TargetResolver) ResolvePath() (string, error) {
	var basePath string
	if tr.Scope == "project" {
		pwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current working directory: %w", err)
		}
		basePath = pwd
	} else {
		// user scope
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		basePath = home
	}

	var agentDir string
	if tr.Agent == "common" {
		agentDir = ".agents"
	} else {
		agentDir = fmt.Sprintf(".%s", tr.Agent) // e.g. .copilot
	}

	// We'll use a unified structure: ~/.agents/skills/ or ~/.copilot/skills/
	// If the user specifies "common", it goes into ~/.agents/skills/
	return filepath.Join(basePath, agentDir, "skills"), nil
}
