package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTargetResolver_ResolvePath(t *testing.T) {
	home, _ := os.UserHomeDir()
	pwd, _ := os.Getwd()

	tests := []struct {
		name     string
		agent    string
		scope    string
		expected string
	}{
		{
			name:     "default user common",
			agent:    "",
			scope:    "",
			expected: filepath.Join(home, ".agents", "skills"),
		},
		{
			name:     "project common",
			agent:    "",
			scope:    "project",
			expected: filepath.Join(pwd, ".agents", "skills"),
		},
		{
			name:     "user specific agent",
			agent:    "copilot",
			scope:    "user",
			expected: filepath.Join(home, ".copilot", "skills"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewTargetResolver(tt.agent, tt.scope)
			got, err := tr.ResolvePath()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}
