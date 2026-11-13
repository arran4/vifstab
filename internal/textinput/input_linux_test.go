//go:build linux

package textinput

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestStableReferenceCannotFollowPathReplacementToAnotherObject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	const original = "# original regular input\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := open(path, func(path string) (metadata, error) {
		m, err := openMetadata(path)
		if err != nil {
			return nil, err
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		// A path-based reopen would block. No device node is created or opened.
		if err := unix.Mkfifo(path, 0600); err != nil {
			t.Fatal(err)
		}
		return m, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	got, err := io.ReadAll(f)
	if err != nil || string(got) != original {
		t.Fatalf("input path replacement redirected the read: %q %v", got, err)
	}
}

func TestRegularSymlinksAndBorrowedPipeInputsRemainUsable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	data := []byte("# fixture\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	link := path + "-link"
	if err := os.Symlink("input", link); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{path, link} {
		got, err := ReadFile(source)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("regular/symlink input: %q %v", got, err)
		}
	}
	after, err := os.Stat(path)
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("read changed fixture inode/mode/mtime: %v", err)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	if err := Check(r); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("borrowed pipe input: %q %v", got, err)
	}
}

func TestNamedPipeInputRetainsBlockingStreamSemantics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fifo")
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() {
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err == nil {
			_, err = f.Write([]byte("# fifo fixture\n"))
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
		}
		finished <- err
	}()
	got, err := ReadFile(path)
	if err != nil || string(got) != "# fifo fixture\n" {
		t.Fatalf("named pipe input: %q %v", got, err)
	}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
}

func TestDescriptorsAreMetadataOnlyThenReadOnlyAndCloseOnExec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	m, err := openMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = m.Close() }()
	fd := m.(*linuxMetadata).file.Fd()
	flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil || flags&unix.O_PATH == 0 {
		t.Fatalf("metadata descriptor opened for data: %x %v", flags, err)
	}
	cloexec, err := unix.FcntlInt(fd, unix.F_GETFD, 0)
	if err != nil || cloexec&unix.FD_CLOEXEC == 0 {
		t.Fatalf("metadata descriptor can leak: %v", err)
	}
	f, err := m.ReadOnly()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	fd = f.(*os.File).Fd()
	flags, err = unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil || flags&unix.O_ACCMODE != unix.O_RDONLY {
		t.Fatalf("input descriptor is writable: %x %v", flags, err)
	}
	cloexec, err = unix.FcntlInt(fd, unix.F_GETFD, 0)
	if err != nil || cloexec&unix.FD_CLOEXEC == 0 {
		t.Fatalf("input descriptor can leak: %v", err)
	}
}

func TestMissingInputPreservesNativeNotExistClassification(t *testing.T) {
	_, err := ReadFile(filepath.Join(t.TempDir(), "new-fstab"))
	if !os.IsNotExist(err) {
		t.Fatalf("a missing input must remain distinguishable from unavailable access: %v", err)
	}
}
