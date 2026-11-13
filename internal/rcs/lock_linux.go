//go:build linux

package rcs

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/arran4/vifstab/internal/textinput"
	"golang.org/x/sys/unix"
)

// LockOSFile holds an advisory per-history lock until Close. The lock pathname
// must remain in place: unlinking it would allow simultaneous locks on two inodes.
// Callers must use a private, trusted backup directory.
func LockOSFile(name string) (io.Closer, error) {
	if info, err := os.Lstat(name); err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("RCS lock must be a private regular file")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	f, err := openOSLock(name)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		_ = f.Close()
		return nil, errors.Join(err, fmt.Errorf("RCS lock must be a private regular file"))
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &historyLock{File: f}, nil
}

// Exclusive creation guarantees a new regular inode. Existing lock objects
// must pass the metadata-only no-follow guard before any data descriptor opens.
func openOSLock(name string) (*os.File, error) {
	fd, err := unix.Open(name, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0600)
	if err == nil {
		return os.NewFile(uintptr(fd), name), nil
	}
	if !errors.Is(err, unix.EEXIST) {
		return nil, err
	}
	checked, err := textinput.OpenRegularNoFollow(name)
	if err != nil {
		return nil, err
	}
	source, ok := checked.(interface{ Fd() uintptr })
	if !ok {
		return nil, errors.Join(fmt.Errorf("RCS lock descriptor unavailable"), checked.Close())
	}
	// Preserve read/write access for locking filesystems that require it. The
	// checked descriptor stays alive until the same inode is reopened; opening
	// the original pathname here would reintroduce the device-replacement race.
	fd, err = unix.Open(fmt.Sprintf("/proc/self/fd/%d", source.Fd()), unix.O_RDWR|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	closeErr := checked.Close()
	if err != nil {
		return nil, errors.Join(err, closeErr)
	}
	f := os.NewFile(uintptr(fd), name)
	if closeErr != nil {
		return nil, errors.Join(closeErr, f.Close())
	}
	return f, nil
}

type historyLock struct{ *os.File }

func (l *historyLock) Close() error {
	err := unix.Flock(int(l.Fd()), unix.LOCK_UN)
	return errors.Join(err, l.File.Close())
}

// ReadOSHistory refuses symlinks and non-regular files before reading bytes.
func ReadOSHistory(name string) ([]byte, error) {
	info, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("RCS history must be a regular file")
	}
	f, err := textinput.OpenRegularNoFollow(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !after.Mode().IsRegular() || !os.SameFile(info, after) {
		return nil, fmt.Errorf("RCS history changed while opening")
	}
	return io.ReadAll(f)
}
