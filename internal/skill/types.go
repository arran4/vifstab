package skill

import "time"

// SkillMetadata represents the provenance and status of an installed skill.
type SkillMetadata struct {
	Name            string    `json:"name"`
	SourceType      string    `json:"source_type"` // "github", "local", "embedded"
	SourceURL       string    `json:"source_url"`  // e.g., "owner/repo" or "/path/to/local"
	SourcePath      string    `json:"source_path"` // path within the repository
	Revision        string    `json:"revision"`    // commit hash or digest
	InstallTime     time.Time `json:"install_time"`
	TargetAgent     string    `json:"target_agent"`
	Scope           string    `json:"scope"`
	LocalModHash    string    `json:"local_mod_hash,omitempty"` // hash of skill directory to detect local modifications
}
