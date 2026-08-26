package systemd

import (
	"fmt"
	"os"
	"os/exec"
)

// FS provides filesystem operations to allow mocking
type FS interface {
	Stat(name string) (os.FileInfo, error)
}

type osFS struct{}

func (osFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }

var DefaultFS FS = osFS{}

// Manager abstracts interaction with the init system / service manager.
type Manager interface {
	IsActive() bool
	Reload() error
}

// RealManager implements Manager for systemd.
type RealManager struct {
	FS FS
}

// NewManager creates a new systemd manager.
func NewManager(fs FS) *RealManager {
	if fs == nil {
		fs = DefaultFS
	}
	return &RealManager{FS: fs}
}

// IsActive returns true if systemd is the active init system.
func (m *RealManager) IsActive() bool {
	info, err := m.FS.Stat("/run/systemd/system")
	return err == nil && info.IsDir()
}

// Reload executes `systemctl daemon-reload`.
func (m *RealManager) Reload() error {
	cmd := exec.Command("systemctl", "daemon-reload")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed: %w", err)
	}
	return nil
}
