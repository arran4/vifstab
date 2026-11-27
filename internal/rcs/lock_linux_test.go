//go:build linux

package rcs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestPersistentNativeLockUsesSamePrivateInodeAndReadWriteAccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.lock")
	first, err := LockOSFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		t.Fatalf("new lock metadata: %v", err)
	}
	second, err := openOSLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Close() }()
	opened, err := second.Stat()
	if err != nil || !os.SameFile(info, opened) {
		t.Fatalf("existing lock reopened a different inode: %v", err)
	}
	flags, err := unix.FcntlInt(second.Fd(), unix.F_GETFL, 0)
	if err != nil || flags&unix.O_ACCMODE != unix.O_RDWR || flags&unix.O_NONBLOCK == 0 {
		t.Fatalf("lock access contract changed: %x %v", flags, err)
	}
	flags, err = unix.FcntlInt(second.Fd(), unix.F_GETFD, 0)
	if err != nil || flags&unix.FD_CLOEXEC == 0 {
		t.Fatalf("lock descriptor can leak to editor: %v", err)
	}
	if err := unix.Flock(int(second.Fd()), unix.LOCK_EX|unix.LOCK_NB); !errors.Is(err, unix.EWOULDBLOCK) {
		t.Fatalf("independent descriptor did not contend on the first lock: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := unix.Flock(int(second.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatalf("released lock unavailable: %v", err)
	}
	if _, err := second.Write([]byte("persistent fixture")); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := LockOSFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := again.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "persistent fixture" {
		t.Fatalf("reopening truncated the persistent lock: %q %v", data, err)
	}
	after, err := os.Stat(path)
	if err != nil || !os.SameFile(info, after) || after.Mode().Perm() != 0600 {
		t.Fatalf("lock was unlinked/replaced: %v", err)
	}
}

func TestNativeHistoryAndLockRejectNonregularFixtures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")
	if err := os.WriteFile(path, []byte("sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	link, fifo := path+"-link", path+"-fifo"
	if err := os.Symlink("history", link); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{link, fifo, dir} {
		if _, err := ReadOSHistory(name); err == nil {
			t.Fatalf("nonregular history accepted: %s", name)
		}
		if lock, err := LockOSFile(name); err == nil {
			_ = lock.Close()
			t.Fatalf("nonregular lock accepted: %s", name)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "sentinel" {
		t.Fatalf("rejected symlink changed target: %q %v", data, err)
	}
}
