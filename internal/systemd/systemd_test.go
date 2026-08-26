package systemd

import (
	"os"
	"testing"
	"time"
)

type mockFS struct {
	statFunc func(name string) (os.FileInfo, error)
}

func (m *mockFS) Stat(name string) (os.FileInfo, error) {
	if m.statFunc != nil {
		return m.statFunc(name)
	}
	return nil, os.ErrNotExist
}

type mockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m mockFileInfo) ModTime() time.Time { return m.modTime }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() any           { return nil }

func TestIsActive(t *testing.T) {
	tests := []struct {
		name     string
		statFunc func(string) (os.FileInfo, error)
		expected bool
	}{
		{
			name: "systemd active",
			statFunc: func(n string) (os.FileInfo, error) {
				if n == "/run/systemd/system" {
					return mockFileInfo{isDir: true}, nil
				}
				return nil, os.ErrNotExist
			},
			expected: true,
		},
		{
			name: "not systemd - file instead of dir",
			statFunc: func(n string) (os.FileInfo, error) {
				if n == "/run/systemd/system" {
					return mockFileInfo{isDir: false}, nil
				}
				return nil, os.ErrNotExist
			},
			expected: false,
		},
		{
			name: "not systemd - missing",
			statFunc: func(n string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			expected: false,
		},
		{
			name: "not systemd - permission error",
			statFunc: func(n string) (os.FileInfo, error) {
				if n == "/run/systemd/system" {
					return nil, os.ErrPermission
				}
				return nil, os.ErrNotExist
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &mockFS{statFunc: tt.statFunc}
			m := NewManager(fs)
			if active := m.IsActive(); active != tt.expected {
				t.Errorf("IsActive() = %v, want %v", active, tt.expected)
			}
		})
	}
}
