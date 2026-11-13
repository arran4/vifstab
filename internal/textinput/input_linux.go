//go:build linux

package textinput

import (
	"fmt"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

type linuxMetadata struct{ file *os.File }

func openMetadata(path string) (metadata, error) {
	// O_PATH acquires an inode reference without opening a device for I/O.
	fd, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return &linuxMetadata{os.NewFile(uintptr(fd), path)}, nil
}

func (m *linuxMetadata) Stat() (fs.FileInfo, error) { return m.file.Stat() }
func (m *linuxMetadata) Close() error               { return m.file.Close() }
func (m *linuxMetadata) ReadOnly() (fs.File, error) {
	// The descriptor names the checked inode even if the original path changes.
	// Normal permissions are checked with the caller's credentials. Keep the
	// original blocking semantics of intentional FIFO/character inputs.
	fd, err := unix.Open(fmt.Sprintf("/proc/self/fd/%d", m.file.Fd()), unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), m.file.Name()), nil
}
