//go:build linux

package diagram

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/diskfs/go-diskfs/backend"
	"golang.org/x/sys/unix"
)

type linuxImageMetadata struct{ file *os.File }

func openImageMetadata(path string) (imageMetadata, error) {
	// O_PATH acquires only a filesystem metadata reference. It does not open
	// the object for I/O or invoke a device driver. Symlinked images still work.
	fd, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return &linuxImageMetadata{file: os.NewFile(uintptr(fd), path)}, nil
}

func (m *linuxImageMetadata) Stat() (fs.FileInfo, error) { return m.file.Stat() }
func (m *linuxImageMetadata) Close() error               { return m.file.Close() }

func (m *linuxImageMetadata) ReadOnly() (backend.File, error) {
	// Reopen this stable, already checked inode, not the possibly replaced
	// pathname. Ordinary read permissions are checked with the caller's context.
	path := fmt.Sprintf("/proc/self/fd/%d", m.file.Fd())
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}
