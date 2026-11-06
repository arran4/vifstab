package diagram

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/partition"
	"github.com/diskfs/go-diskfs/partition/gpt"
)

type testImageMetadata struct {
	info     fs.FileInfo
	statErr  error
	readErr  error
	closeErr error
	reader   backend.File
	reads    int
	closes   int
}

func TestImageMalformedTableAndBoundsDoNotPanic(t *testing.T) {
	var nilTable *gpt.Table
	for _, tc := range []struct {
		name  string
		table partition.Table
		size  int64
	}{
		{"nil typed table", nilTable, 1024},
		{"zero image size", &gpt.Table{}, 0},
		{"nil partition", &gpt.Table{Partitions: []*gpt.Partition{nil}}, 1024},
		{"negative wrapped start", &gpt.Table{Partitions: []*gpt.Partition{{Start: ^uint64(0), Size: 1}}}, 1024},
		{"outside image", &gpt.Table{Partitions: []*gpt.Partition{{Start: 3, Size: 1}}}, 1024},
		{"size exceeds image", &gpt.Table{Partitions: []*gpt.Partition{{Start: 1, Size: 1024}}}, 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := Print(io.Discard, "image", Dependencies{OS: sourceOS{info: sourceInfo(t, 0)}, OpenImage: func(string, ...diskfs.OpenOpt) (*disk.Disk, error) {
				return &disk.Disk{Table: tc.table, Size: tc.size}, nil
			}})
			if err == nil {
				t.Fatal("malformed image model accepted")
			}
		})
	}
}

func (m *testImageMetadata) Stat() (fs.FileInfo, error)      { return m.info, m.statErr }
func (m *testImageMetadata) ReadOnly() (backend.File, error) { m.reads++; return m.reader, m.readErr }
func (m *testImageMetadata) Close() error                    { m.closes++; return m.closeErr }

type testImageReader struct {
	*os.File
	info    fs.FileInfo
	statErr error
	closes  int
}

func (r *testImageReader) Stat() (fs.FileInfo, error) { return r.info, r.statErr }
func (r *testImageReader) Close() error               { r.closes++; return nil }

func TestImageGuardRejectsNonregularMetadataBeforeDataOpen(t *testing.T) {
	for _, mode := range []fs.FileMode{fs.ModeDevice, fs.ModeDevice | fs.ModeCharDevice, fs.ModeNamedPipe, fs.ModeDir, fs.ModeSymlink} {
		m := &testImageMetadata{info: sourceInfo(t, mode)}
		m.readErr = errors.New("a data open would violate the metadata-only boundary")
		_, err := regularImageFile("fixture", func(string) (imageMetadata, error) { return m, nil })
		if err == nil || m.reads != 0 || m.closes != 1 {
			t.Fatalf("nonregular metadata opened for data: %v reads%d closes%d", err, m.reads, m.closes)
		}
	}
}

func TestImageGuardMissingHandles(t *testing.T) {
	if _, err := regularImageFile("fixture", func(string) (imageMetadata, error) { return nil, nil }); err == nil {
		t.Fatal("missing metadata handle accepted")
	}
	m := &testImageMetadata{info: sourceInfo(t, 0)}
	if _, err := regularImageFile("fixture", func(string) (imageMetadata, error) { return m, nil }); err == nil || m.closes != 1 {
		t.Fatalf("missing data handle not cleaned up: %v", err)
	}
}

func TestImageDiagramWriterErrors(t *testing.T) {
	for _, short := range []bool{false, true} {
		err := Print(failingDiagramWriter{short: short}, "image", Dependencies{OS: sourceOS{info: sourceInfo(t, 0)}, OpenImage: func(string, ...diskfs.OpenOpt) (*disk.Disk, error) {
			return &disk.Disk{Table: &gpt.Table{}, Size: 1024}, nil
		}})
		want := io.ErrClosedPipe
		if short {
			want = io.ErrShortWrite
		}
		if !errors.Is(err, want) {
			t.Fatalf("image output failure lost: %v", err)
		}
	}
}

func TestImageGuardFailureCleanup(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "regular-image")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	other, err := os.CreateTemp(t.TempDir(), "other")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = other.Close() }()
	otherInfo, err := other.Stat()
	if err != nil {
		t.Fatal(err)
	}
	fault := errors.New("injected image fault")
	for _, tc := range []struct {
		name                        string
		edit                        func(*testImageMetadata, *testImageReader)
		wantReads, wantReaderCloses int
	}{
		{"stat unavailable", func(m *testImageMetadata, r *testImageReader) { m.statErr = fault }, 0, 0},
		{"missing stat information", func(m *testImageMetadata, r *testImageReader) { m.info = nil }, 0, 0},
		{"read access unavailable", func(m *testImageMetadata, r *testImageReader) { m.readErr = fault }, 1, 0},
		{"opened stat unavailable", func(m *testImageMetadata, r *testImageReader) { r.statErr = fault }, 1, 1},
		{"different opened inode", func(m *testImageMetadata, r *testImageReader) { r.info = otherInfo }, 1, 1},
		{"opened nonregular", func(m *testImageMetadata, r *testImageReader) { r.info = sourceInfo(t, fs.ModeDevice) }, 1, 1},
		{"metadata close failure", func(m *testImageMetadata, r *testImageReader) { m.closeErr = fault }, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &testImageReader{File: file, info: info}
			m := &testImageMetadata{info: info, reader: r}
			tc.edit(m, r)
			_, err := regularImageFile("fixture", func(string) (imageMetadata, error) { return m, nil })
			if err == nil || m.reads != tc.wantReads || r.closes != tc.wantReaderCloses || m.closes != 1 {
				t.Fatalf("cleanup err=%v reads%d readerCloses%d metadataCloses%d", err, m.reads, r.closes, m.closes)
			}
		})
	}
}
