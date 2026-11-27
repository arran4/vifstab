//go:build linux

package diagram

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/partition/mbr"
	"golang.org/x/sys/unix"
)

func TestImageReopenUsesStableInodeAfterPathReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image")
	if err := os.WriteFile(path, []byte("original regular file"), 0600); err != nil {
		t.Fatal(err)
	}
	reader, err := regularImageFile(path, func(path string) (imageMetadata, error) {
		metadata, err := openImageMetadata(path)
		if err != nil {
			return nil, err
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		// A path-based reopen would block or open the wrong object. The stable
		// metadata reference must keep the original regular inode instead.
		if err := unix.Mkfifo(path, 0600); err != nil {
			t.Fatal(err)
		}
		return metadata, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	data, err := io.ReadAll(reader)
	if err != nil || string(data) != "original regular file" {
		t.Fatalf("reopened replaced path: %q %v", data, err)
	}
}

func TestImageMetadataOnFIFOIsNonblockingAndNeverOpensData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fifo")
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := regularImageFile(path, openImageMetadata); err == nil {
		t.Fatal("FIFO accepted as an image")
	}
}

func TestNativeImageHandlesAreMetadataOnlyThenReadOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	metadata, err := openImageMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = metadata.Close() }()
	fd := metadata.(*linuxImageMetadata).file.Fd()
	flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil || flags&unix.O_PATH == 0 {
		t.Fatalf("metadata object opened for data: flags%x %v", flags, err)
	}
	cloexec, err := unix.FcntlInt(fd, unix.F_GETFD, 0)
	if err != nil || cloexec&unix.FD_CLOEXEC == 0 {
		t.Fatalf("metadata fd can leak to subprocess: %v", err)
	}
	info, err := metadata.Stat()
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("regular image not classified: %v", err)
	}
	reader, err := metadata.ReadOnly()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	dataFD := reader.(*os.File).Fd()
	flags, err = unix.FcntlInt(dataFD, unix.F_GETFL, 0)
	if err != nil || flags&unix.O_ACCMODE != unix.O_RDONLY {
		t.Fatalf("data handle writable: flags%x %v", flags, err)
	}
	cloexec, err = unix.FcntlInt(dataFD, unix.F_GETFD, 0)
	if err != nil || cloexec&unix.FD_CLOEXEC == 0 {
		t.Fatalf("data fd can leak to subprocess: %v", err)
	}
}

func TestRealRegularImagesRemainReadOnlyAndSupportSymlinks(t *testing.T) {
	for _, tableType := range []string{"mbr", "gpt"} {
		t.Run(tableType, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "disk.img")
			f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.Truncate(4 * 1024 * 1024); err != nil {
				t.Fatal(err)
			}
			d, err := diskfs.OpenBackend(file.New(f, false))
			if err != nil {
				t.Fatal(err)
			}
			if tableType == "mbr" {
				err = d.Partition(&mbr.Table{LogicalSectorSize: 512, PhysicalSectorSize: 512, Partitions: []*mbr.Partition{{Start: 2048, Size: 2048, Type: mbr.Linux}}})
			} else {
				err = d.Partition(&gpt.Table{LogicalSectorSize: 512, PhysicalSectorSize: 512, Partitions: []*gpt.Partition{{Index: 1, Start: 2048, End: 4095, Size: 1024 * 1024, Type: gpt.LinuxFilesystem, Name: "fixture"}}})
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := d.Close(); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			beforeInfo, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(filepath.Dir(path), "image-link")
			if err := os.Symlink("disk.img", link); err != nil {
				t.Fatal(err)
			}
			for _, source := range []string{path, link} {
				var output bytes.Buffer
				if err := Print(&output, source); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(output.String(), "Partition Table: "+tableType) || !strings.Contains(output.String(), "Partition 1: Start 1048576, Size 1048576") {
					t.Fatalf("regular image parsing changed: %s", output.String())
				}
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			afterInfo, err := os.Stat(path)
			if err != nil || sha256.Sum256(before) != sha256.Sum256(after) || !os.SameFile(beforeInfo, afterInfo) || beforeInfo.Mode() != afterInfo.Mode() || !beforeInfo.ModTime().Equal(afterInfo.ModTime()) {
				t.Fatalf("image mutated: %v", err)
			}
		})
	}
}

func TestMalformedRegularImagesFailWithoutPanicOrWrites(t *testing.T) {
	for _, data := range [][]byte{nil, {0}, make([]byte, 512), make([]byte, 4096)} {
		path := filepath.Join(t.TempDir(), "malformed.img")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		if err := Print(&output, path); err == nil {
			t.Fatal("malformed image accepted")
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(data, after) {
			t.Fatalf("malformed image modified: %v", err)
		}
	}
}

func TestImageGuardReturnsOnlyVerifiedRegularReader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image")
	if err := os.WriteFile(path, []byte("regular fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	reader, err := regularImageFile(path, openImageMetadata)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	data, err := io.ReadAll(reader)
	if err != nil || !bytes.Equal(data, []byte("regular fixture")) {
		t.Fatalf("reader bytes=%q %v", data, err)
	}
}
